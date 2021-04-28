// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

package dirwatcher

import (
	"bufio"
	"container/heap"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"gitlab.com/lightmeter/controlcenter/logeater/announcer"
	"gitlab.com/lightmeter/controlcenter/pkg/postfix"
	parser "gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser"
	"gitlab.com/lightmeter/controlcenter/util/closeutil"
	"gitlab.com/lightmeter/controlcenter/util/errorutil"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"time"
)

type fileEntry struct {
	filename         string
	modificationTime time.Time
}

type fileEntryList []fileEntry

type fileQueues map[string]fileEntryList

func sortedEntriesFilteredByPatternAndMoreRecentThanTime(list fileEntryList, r filenameRecognizer, initialTime time.Time) fileEntryList {
	entries := make(fileEntryList, 0, len(list))

	type rec struct {
		entry fileEntry
		index int
	}

	recs := []rec{}

	for _, entry := range list {
		basename := path.Base(entry.filename)
		matches := r.reg.FindSubmatch([]byte(basename))

		if len(matches) == 0 {
			continue
		}

		// always include the most current log file as, if it's older than the initial time
		// it must be present when we watch for changes on it as they potentially arrive in the future.
		// This relates to #309
		if basename != r.pattern && entry.modificationTime.Before(initialTime) {
			continue
		}

		index := func() int {
			if len(matches[3]) == 0 {
				return 0
			}

			index, err := strconv.Atoi(string(matches[3]))
			errorutil.MustSucceed(err, "Atoi")

			return index
		}()

		recs = append(recs, rec{entry: entry, index: index})
	}

	sort.Slice(recs, func(i, j int) bool {
		// the base file is **always** the last element in the list
		if path.Base(recs[j].entry.filename) == r.pattern {
			return true
		}

		if path.Base(recs[i].entry.filename) == r.pattern {
			return false
		}

		return recs[i].index*int(r.order) < recs[j].index*int(r.order)
	})

	for _, r := range recs {
		entries = append(entries, r.entry)
	}

	return entries
}

type filenameSortOrder int

const (
	filenameReverseOrder filenameSortOrder = -1
	filenameNormalOrder  filenameSortOrder = 1
)

type filenameRecognizerBuilder struct {
	builder func(string) string
	order   filenameSortOrder
}

type filenameRecognizer struct {
	reg     *regexp.Regexp
	pattern string
	order   filenameSortOrder
}

func (f filenameRecognizerBuilder) build(pattern string) filenameRecognizer {
	r, err := regexp.Compile(f.builder(pattern))

	errorutil.MustSucceed(err, `trying to build regexp for pattern "`+pattern+`"`)

	return filenameRecognizer{
		reg:     r,
		pattern: pattern,
		order:   f.order,
	}
}

// NOTE: More information about how logrotate defines the filename conventions at:
// http://man7.org/linux/man-pages/man5/logrotate.conf.5.html
var filenameRecognizers = []filenameRecognizerBuilder{
	{
		builder: func(pattern string) string {
			// format mail.log-20201008.gz, where the suffix is a date, lexicographically sortable.
			return `^(` + pattern + `)(-(\d{8})(\.gz)?)?$`
		},
		order: filenameNormalOrder,
	},

	{
		builder: func(pattern string) string {
			// format mail.log.3.gz
			// the higher the suffix value, the older the file is.
			return `^(` + pattern + `)(\.(\d+)(\.gz)?)?$`
		},
		order: filenameReverseOrder,
	},
}

func buildFilesToImport(list fileEntryList, patterns LogPatterns, initialTime time.Time) fileQueues {
	queuesMatchAFileSuffixConvention := func(queues fileQueues) bool {
		for _, queue := range queues {
			// there must be at least one with suffix for the suffix convention to be recognized
			// plus possibly the base file, without the suffix.
			if len(queue) > 1 {
				return true
			}
		}

		return false
	}

	var queues fileQueues

	for _, f := range filenameRecognizers {
		queues = buildFilesToImportByPatternKind(list, patterns, f, initialTime)

		if queuesMatchAFileSuffixConvention(queues) {
			return queues
		}
	}

	// bail out and use whatever we've been able to find,
	// which can be correct in case it's not possible to detect a suffix
	// (for instance, there are only files without suffix: mail.log, mail.warn and so on...)
	return queues
}

func buildFilesToImportByPatternKind(list fileEntryList, patterns LogPatterns, r filenameRecognizerBuilder, initialTime time.Time) fileQueues {
	if len(patterns.patterns) == 0 {
		return fileQueues{}
	}

	queues := make(fileQueues, len(patterns.patterns))

	for _, pattern := range patterns.patterns {
		r := r.build(pattern)
		queues[pattern] = sortedEntriesFilteredByPatternAndMoreRecentThanTime(list, r, initialTime)
	}

	return queues
}

// Given a leap year, what nth second is a time on it?
func secondInTheYear(month time.Month, day, hour, minute, second int) float64 {
	asRefTime := func(month time.Month, day, hour, minute, second int) time.Time {
		return time.Date(2000, month, day, hour, minute, second, 0, time.UTC)
	}

	return asRefTime(month, day, hour, minute, second).
		Sub(asRefTime(time.January, 1, 0, 0, 0)).
		Seconds()
}

func readFirstLine(scanner *bufio.Scanner) (parser.Time, bool, error) {
	if !scanner.Scan() {
		// empty file
		return parser.Time{}, false, nil
	}

	// read first line
	h1, _, err := parser.Parse(scanner.Bytes())

	return h1.Time, true, func() error {
		if err == nil {
			return nil
		}

		return errorutil.Wrap(err)
	}()
}

func readLastLine(scanner *bufio.Scanner) (parser.Time, bool, error) {
	lastLine := ""

	linesRead := 0

	for scanner.Scan() {
		linesRead++

		lastLine = string(scanner.Bytes())
	}

	if linesRead == 0 {
		return parser.Time{}, false, nil
	}

	// reached the last line
	h2, _, err := parser.Parse([]byte(lastLine))

	return h2.Time, true, func() error {
		if err == nil {
			return nil
		}

		return errorutil.Wrap(err)
	}()
}

func guessInitialDateForFile(reader io.Reader, originalModificationTime time.Time) (time.Time, error) {
	modificationTime := originalModificationTime.In(time.UTC)

	scanner := bufio.NewScanner(reader)

	timeFirstLine, ok, err := readFirstLine(scanner)

	if !ok {
		// empty file
		return modificationTime, nil
	}

	if !parser.IsRecoverableError(err) {
		// failed to read first line
		return time.Time{}, errorutil.Wrap(err)
	}

	secondsInYearFirstLine := secondInTheYear(
		timeFirstLine.Month,
		int(timeFirstLine.Day),
		int(timeFirstLine.Hour),
		int(timeFirstLine.Minute),
		int(timeFirstLine.Second))

	timeLastLine, ok, err := readLastLine(scanner)

	secondsInTheYearForTime := func(t time.Time) float64 {
		return secondInTheYear(t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	}

	// as we normalize the file modification time to UTC, the original time might be
	// up to 12 hours ahead, when ignoring the timezone
	modificationTimePlus12Hours := modificationTime.Add(time.Hour * 12)

	secondsInYearModificationTime := secondsInTheYearForTime(modificationTime)
	secondsInYearModificationTimePlus12Hours := secondsInTheYearForTime(modificationTimePlus12Hours)

	adjustYearOffsetAfter12HoursJumpForward := modificationTimePlus12Hours.Year() - modificationTime.Year()

	if !ok {
		// one line file
		computeYear := func(a, b float64) int {
			yearOffset := 0

			if a < b {
				yearOffset++
			}

			return modificationTime.Year() - yearOffset
		}

		year := computeYear(secondsInYearModificationTime, secondsInYearFirstLine) + adjustYearOffsetAfter12HoursJumpForward

		return timeFirstLine.Time(year, modificationTime.Location()), nil
	}

	if !parser.IsRecoverableError(err) {
		// failed reading last line
		return time.Time{}, errorutil.Wrap(err)
	}

	secondsInYearLastLine := secondInTheYear(
		timeLastLine.Month,
		int(timeLastLine.Day),
		int(timeLastLine.Hour),
		int(timeLastLine.Minute),
		int(timeLastLine.Second))

	ordered := func(a, b, c float64) bool {
		return a <= b && b <= c
	}

	basicOffset := func(begin, end, modified float64) int {
		if begin <= end && end <= modified {
			return 0
		}

		return 1
	}

	yearOffset := func() int {
		switch {
		case ordered(secondsInYearFirstLine, secondsInYearModificationTime, secondsInYearLastLine):
			fallthrough
		case ordered(secondsInYearModificationTime, secondsInYearFirstLine, secondsInYearLastLine):
			return basicOffset(secondsInYearFirstLine, secondsInYearLastLine, secondsInYearModificationTimePlus12Hours) - adjustYearOffsetAfter12HoursJumpForward
		default:
			return basicOffset(secondsInYearFirstLine, secondsInYearLastLine, secondsInYearModificationTime)
		}
	}()

	year := modificationTime.Year() - yearOffset

	return timeFirstLine.Time(year, modificationTime.Location()), nil
}

type fileDescriptor struct {
	modificationTime time.Time
	reader           fileReader
}

var ErrEmptyFileList = errors.New(`No valid log files found`)

func findEarlierstTimeFromFiles(files []fileDescriptor) (time.Time, error) {
	if len(files) == 0 {
		return time.Time{}, errorutil.Wrap(ErrEmptyFileList)
	}

	var t time.Time

	// NOTE: this code does not work for files from before the Unix epoch
	for _, file := range files {
		ft, err := guessInitialDateForFile(file.reader, file.modificationTime)

		if err != nil {
			return time.Time{}, errorutil.Wrap(err)
		}

		if (ft.Before(t) || t == time.Time{}) {
			t = ft
		}
	}

	return t, nil
}

func FindInitialLogTime(content DirectoryContent, patterns LogPatterns) (time.Time, error) {
	queues, err := buildQueuesForDirImporter(content, patterns, time.Time{})

	if err != nil {
		return time.Time{}, errorutil.Wrap(err)
	}

	descriptors := []fileDescriptor{}

	closers := closeutil.New()

	defer func() {
		errorutil.MustSucceed(closers.Close())
	}()

	for _, queue := range queues {
		if len(queue) == 0 {
			continue
		}

		entry := queue[0]

		filename := entry.filename

		reader, err := content.readerForEntry(filename)

		if err != nil {
			return time.Time{}, errorutil.Wrap(err)
		}

		closer := closeutil.ConvertToCloser(func() error {
			err := reader.Close()
			if err != nil {
				return fmt.Errorf("could not close file: %v, %w", filename, err)
			}
			return nil
		})
		closers.Add(closer)

		d := fileDescriptor{modificationTime: entry.modificationTime, reader: reader}

		descriptors = append(descriptors, d)
	}

	return findEarlierstTimeFromFiles(descriptors)
}

type fileReader interface {
	io.Reader
	io.Closer
}

type fileReadSeeker interface {
	fileReader
	io.Seeker
}

type fileWatcher interface {
	run(onNewRecord func(parser.Header, parser.Payload))
}

type DirectoryContent interface {
	dirName() string
	fileEntries() fileEntryList
	modificationTimeForEntry(filename string) (time.Time, error)
	readerForEntry(filename string) (fileReader, error)
	watcherForEntry(filename string, offset int64) (fileWatcher, error)
	readSeekerForEntry(filename string) (fileReadSeeker, error)
}

type DirectoryImporter struct {
	content     DirectoryContent
	pub         postfix.Publisher
	announcer   announcer.ImportAnnouncer
	initialTime time.Time
	patterns    LogPatterns
}

func NewDirectoryImporter(
	content DirectoryContent,
	pub postfix.Publisher,
	announcer announcer.ImportAnnouncer,
	initialTime time.Time,
	patterns LogPatterns,
) DirectoryImporter {
	return DirectoryImporter{content, pub, announcer, initialTime, patterns}
}

var ErrLogFilesNotFound = errors.New("Could not find any matching log files")

func buildQueuesForDirImporter(content DirectoryContent, patterns LogPatterns, initialTime time.Time) (fileQueues, error) {
	onError := func() (fileQueues, error) {
		return fileQueues{}, errorutil.Wrap(ErrLogFilesNotFound, "Could not find any matching log files in the directory: ", content.dirName(), " that are more recent than ", initialTime)
	}

	entries := content.fileEntries()

	if len(entries) == 0 {
		return onError()
	}

	queues := buildFilesToImport(entries, patterns, initialTime)

	if len(queues) == 0 {
		return onError()
	}

	for _, q := range queues {
		if len(q) != 0 {
			return queues, nil
		}
	}

	return onError()
}

type LogPatterns struct {
	patterns []string
	indexes  map[string]int
}

func BuildLogPatterns(patterns []string) LogPatterns {
	indexes := map[string]int{}

	for i, v := range patterns {
		indexes[v] = i
	}

	return LogPatterns{patterns: patterns, indexes: indexes}
}

var DefaultLogPatterns = BuildLogPatterns([]string{"mail.log", "mail.err", "mail.warn", "zimbra.log", "maillog"})

type timeConverterChan chan *parser.TimeConverter

type queueProcessor struct {
	readers       []fileReader
	scanners      []*bufio.Scanner
	entries       fileEntryList
	record        postfix.Record
	currentIndex  int
	converter     *parser.TimeConverter
	converterChan timeConverterChan
	pattern       string
	filename      string
	line          uint64
}

type limitedFileReader struct {
	reader fileReader
	io.LimitedReader
}

func (l *limitedFileReader) Close() error {
	return l.reader.Close()
}

func buildLimitedFileReader(reader fileReader, offset int64) fileReader {
	r := limitedFileReader{
		reader:        reader,
		LimitedReader: io.LimitedReader{R: reader, N: offset},
	}

	return &r
}

func buildReaderForCurrentEntry(content DirectoryContent, entry fileEntry) (fileReader, int64, error) {
	readSeeker, err := content.readSeekerForEntry(entry.filename)

	if err != nil {
		return nil, 0, errorutil.Wrap(err)
	}

	offset, err := readSeeker.Seek(0, io.SeekEnd)

	defer func() {
		if err != nil {
			errorutil.MustSucceed(readSeeker.Close(), "Closing on seeking file to end")
		}
	}()

	if err != nil {
		return nil, 0, errorutil.Wrap(err)
	}

	_, err = readSeeker.Seek(0, io.SeekStart)

	if err != nil {
		return nil, 0, errorutil.Wrap(err)
	}

	reader := buildLimitedFileReader(readSeeker, offset)

	return reader, offset, nil
}

func buildReaderAndScannerForEntry(offsetChan chan int64, content DirectoryContent, pattern string, entry fileEntry) (fileReader, *bufio.Scanner, error) {
	if path.Base(entry.filename) == pattern {
		// special case: current log file, that is being updated by postfix on a different process
		// NOTE: Yes, this is a race condition.
		// here we create a reader that will read the file up to the that point,
		// even if new data is written in the file in the meanwhile, as such new lines
		// will be read by another thread, the "file watcher".
		// Once we have such offset, we notify the file watcher about where to start reading
		reader, offset, err := buildReaderForCurrentEntry(content, entry)

		if err != nil {
			return nil, nil, errorutil.Wrap(err)
		}

		// inform the "watch current log file" thread about the offset
		// in the file it should start watching from
		offsetChan <- offset

		return reader, bufio.NewScanner(reader), nil
	}

	reader, err := content.readerForEntry(entry.filename)

	if err != nil {
		return nil, nil, errorutil.Wrap(err)
	}

	return reader, bufio.NewScanner(reader), nil
}

func processorForQueue(offsetChan chan int64, converterChan timeConverterChan, content DirectoryContent, pattern string, entries fileEntryList) (queueProcessor, error) {
	readers := []fileReader{}
	scanners := []*bufio.Scanner{}

	for _, entry := range entries {
		reader, scanner, err := buildReaderAndScannerForEntry(offsetChan, content, pattern, entry)

		if err != nil {
			return queueProcessor{}, errorutil.Wrap(err)
		}

		scanners = append(scanners, scanner)
		readers = append(readers, reader)
	}

	return queueProcessor{
		readers:       readers,
		scanners:      scanners,
		currentIndex:  0,
		converter:     nil,
		converterChan: converterChan,
		entries:       entries,
		pattern:       pattern,
	}, nil
}

func buildQueueProcessors(
	offsetChans map[string]chan int64,
	converterChans map[string]timeConverterChan,
	content DirectoryContent,
	queues fileQueues,
	patterns LogPatterns,
) ([]*queueProcessor, error) {
	p := make([]*queueProcessor, len(queues))

	for k, v := range queues {
		converterChan, ok := converterChans[k]

		if !ok {
			panic("SPANK SPANK SPANK fix your code")
		}

		offsetChan, ok := offsetChans[k]

		if !ok {
			panic("SPANK SPANK SPANK fix your code")
		}

		processor, err := processorForQueue(offsetChan, converterChan, content, k, v)

		if err != nil {
			return []*queueProcessor{}, errorutil.Wrap(err)
		}

		index := patterns.indexes[k]

		p[index] = &processor
	}

	return p, nil
}

func createConverterForQueueProcessor(p *queueProcessor, content DirectoryContent, header parser.Header) (*parser.TimeConverter, error) {
	modificationTime, err := content.modificationTimeForEntry(p.entries[p.currentIndex].filename)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	reader, err := content.readerForEntry(p.entries[p.currentIndex].filename)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	defer func() {
		errorutil.MustSucceed(reader.Close(), "Closing first file in queue")
	}()

	initialTime, err := guessInitialDateForFile(reader, modificationTime)

	if err != nil {
		return nil, errorutil.Wrap(err)
	}

	converter := parser.NewTimeConverter(
		time.Date(initialTime.Year(),
			header.Time.Month,
			int(header.Time.Day),
			int(header.Time.Hour),
			int(header.Time.Minute),
			int(header.Time.Second),
			0,
			initialTime.Location(),
		),
		func(year int, from parser.Time, to parser.Time) {
			log.Info().Msgf("Changed Year to %v (from %v to %v), on log file: %v:%v",
				year, from, to,
				p.record.Location.Filename, p.record.Location.Line)
		})

	// workaround, make converter escape to the heap
	return &converter, nil
}

func setFileLocationOnQueueProcessorIfNeeded(p *queueProcessor) {
	if p.currentIndex >= len(p.readers) {
		return
	}

	if p.line > 0 {
		return
	}

	p.filename = path.Base(p.entries[p.currentIndex].filename)

	log.Info().Msgf("Starting importing log file: %v", p.filename)
}

func updateQueueProcessor(p *queueProcessor, content DirectoryContent, progressNotifier *announcer.Notifier) (bool, error) {
	setFileLocationOnQueueProcessorIfNeeded(p)

	// tries to read something from the queue, ignoring it on the next iteration
	// if nothing is left to be read
	for {
		thereAreFilesToBeProcessed := p.currentIndex < len(p.readers)

		if !thereAreFilesToBeProcessed {
			return false, nil
		}

		scanner := p.scanners[p.currentIndex]

		if !scanner.Scan() {
			// file ended, use next one
			if err := p.readers[p.currentIndex].Close(); err != nil {
				return false, errorutil.Wrap(err)
			}

			log.Info().Msgf("Finished importing log file: %v", p.filename)

			// use last time computed
			progressNotifier.Step(p.record.Time)

			p.currentIndex++
			p.line = 0

			setFileLocationOnQueueProcessorIfNeeded(p)

			// moves to the next file in the queue
			continue
		}

		p.line++

		loc := postfix.RecordLocation{
			Filename: p.filename,
			Line:     p.line,
		}

		// Successfully read
		header, payload, err := parser.Parse(scanner.Bytes())

		if !parser.IsRecoverableError(err) {
			log.Warn().Msgf("Could not parse log line in %v", loc)
			continue
		}

		if p.converter == nil {
			converter, err := createConverterForQueueProcessor(p, content, header)

			if err != nil {
				return false, errorutil.Wrap(err)
			}

			p.converter = converter
		}

		convertedTime := p.converter.Convert(header.Time)

		p.record = postfix.Record{
			Header:   header,
			Payload:  payload,
			Time:     convertedTime,
			Location: loc,
		}

		return true, nil
	}
}

func updateQueueProcessors(content DirectoryContent, processors []*queueProcessor, updatedProcessors *[]*queueProcessor, toBeUpdated int, progressNotifier *announcer.Notifier) error {
	for i, p := range processors {
		isFirstExecution := toBeUpdated != -1

		if isFirstExecution && i != toBeUpdated {
			*updatedProcessors = append(*updatedProcessors, p)
			continue
		}

		shouldKeepProcessor, err := updateQueueProcessor(p, content, progressNotifier)
		if err != nil {
			return errorutil.Wrap(err)
		}

		if shouldKeepProcessor {
			*updatedProcessors = append(*updatedProcessors, p)
			continue
		}

		if p.converter != nil {
			p.converterChan <- p.converter
			continue
		}

		if len(p.entries) == 0 {
			continue
		}

		entry := p.entries[len(p.entries)-1]

		// If there were entries to be processed, but a time converter has not been yet created,
		// create one using the modification time of the most recent file in the queue
		converter := parser.NewTimeConverter(entry.modificationTime,
			func(year int, from parser.Time, to parser.Time) {
				log.Info().Msgf("Changed Year to %v (from %v to %v), on log file: %v",
					year, from, to, entry.filename)
			})

		p.converter = &converter
		p.converterChan <- p.converter
	}

	return nil
}

func chooseIndexForOldestElement(queueProcessors []*queueProcessor) int {
	chosenIndex := -1

	for i, p := range queueProcessors {
		if chosenIndex == -1 || queueProcessors[chosenIndex].record.Time.After(p.record.Time) {
			chosenIndex = i
		}
	}

	if chosenIndex == -1 {
		panic("BUG: your algorithm sucks!")
	}

	return chosenIndex
}

func countNumberOfFilesInQueues(queues fileQueues) int {
	numberOfFiles := 0

	for _, q := range queues {
		numberOfFiles += len(q)
	}

	return numberOfFiles
}

// Open all log files, including archived (compressed or not, but logrotate)
// and read them line by line, publishing them in the right order they were generated (or
// close enough, as the lines have only precision of second, so it's not a "stable sort"),
// so the order among different lines on the same second is not deterministic.
func importExistingLogs(
	offsetChans map[string]chan int64,
	converterChans map[string]timeConverterChan,
	content DirectoryContent,
	queues fileQueues,
	pub postfix.Publisher,
	initialTime time.Time,
	importAnnouncer announcer.ImportAnnouncer,
	patterns LogPatterns,
) error {
	initialImportTime := time.Now()

	queueProcessors, err := buildQueueProcessors(offsetChans, converterChans, content, queues, patterns)
	if err != nil {
		return errorutil.Wrap(err)
	}

	progressNotifier := announcer.NewNotifier(importAnnouncer, countNumberOfFilesInQueues(queues))

	currentLogTime := time.Time{}

	toBeUpdated := -1

	updatedProcessors := make([]*queueProcessor, 0, len(queueProcessors))

	for {
		updatedProcessors = updatedProcessors[0:0]

		err := updateQueueProcessors(content, queueProcessors, &updatedProcessors, toBeUpdated, &progressNotifier)
		if err != nil {
			return errorutil.Wrap(err)
		}

		if len(updatedProcessors) == 0 {
			elapsedTime := time.Since(initialImportTime)

			progressNotifier.End(currentLogTime)

			log.Info().Msgf("Finished importing postfix log directory in: %v", elapsedTime)

			return nil
		}

		queueProcessors = queueProcessors[0:0]
		queueProcessors = append(queueProcessors, updatedProcessors...)

		toBeUpdated = chooseIndexForOldestElement(queueProcessors)

		t := queueProcessors[toBeUpdated].record

		if !t.Time.After(initialTime) {
			continue
		}

		if currentLogTime.IsZero() {
			importAnnouncer.AnnounceStart(t.Time)
		}

		currentLogTime = t.Time

		pub.Publish(t)
	}
}

type newLogsPublisher struct {
	// a temporary buffer for the new lines that arrive before the archived logs are imported
	// so we publish them in chronological order
	records chan postfix.Record
}

func (pub newLogsPublisher) Publish(r postfix.Record) {
	pub.records <- r
}

type sortableRecord struct {
	record parsedRecord
	time   time.Time
}

// Compare lexicographically
func (r sortableRecord) Less(other sortableRecord) bool {
	if r.time.Before(other.time) {
		return true
	}

	if r.time.After(other.time) {
		return false
	}

	if r.record.queueIndex < other.record.queueIndex {
		return true
	}

	if r.record.queueIndex > other.record.queueIndex {
		return false
	}

	return r.record.sequence < other.record.sequence
}

// Implement heap.Interface
type sortableRecordHeap []sortableRecord

func (t sortableRecordHeap) Len() int {
	return len(t)
}

func (t sortableRecordHeap) Less(i, j int) bool {
	return t[i].Less(t[j])
}

func (t sortableRecordHeap) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t *sortableRecordHeap) Push(x interface{}) {
	*t = append(*t, x.(sortableRecord))
}

func (t *sortableRecordHeap) Pop() interface{} {
	old := *t
	n := len(old)
	x := old[n-1]
	*t = old[0 : n-1]

	return x
}

type parsedRecord struct {
	header  parser.Header
	payload parser.Payload

	// When the same queue adds multiple items to the heap that happen in the same second
	// we want to preserve their original order
	// so we use extra values for sorting
	queueIndex int
	sequence   uint64

	loc postfix.RecordLocation
}

// responsible for watching for new logs added to a file
// and buffering them into a channel (outChan).
func startWatchingOnQueue(
	entry fileEntry,
	queueIndex int,
	offsetChan <-chan int64,
	content DirectoryContent,
	outChan chan<- parsedRecord) {
	offset := <-offsetChan

	watcher, err := content.watcherForEntry(entry.filename, offset)

	errorutil.MustSucceed(err, "File watcher for: "+entry.filename)

	sequence := uint64(0)

	watcher.run(func(h parser.Header, p parser.Payload) {
		record := parsedRecord{
			header:     h,
			payload:    p,
			queueIndex: queueIndex,
			sequence:   sequence,
		}

		outChan <- record

		sequence++
	})

	close(outChan)
}

// given the (bufferized) logs received by startWatchingOnQueue() via a channel,
// wait until the initial import is finished, obtaining the time converter from it.
//
// While the initial import happens,
// the time converter is not available,
// being owned by the import process.
// once it's finished, we take ownership
// over the converter and start using it from
// the exact point in time the import stopped.
func startTimestampingParsedLogs(
	converterChan timeConverterChan,
	sortableRecordsChan chan<- sortableRecord,
	parsedRecordsChan <-chan parsedRecord,
	done chan<- struct{}) {
	converter := <-converterChan

	for p := range parsedRecordsChan {
		t := converter.Convert(p.header.Time)
		r := sortableRecord{record: p, time: t}
		sortableRecordsChan <- r
	}

	done <- struct{}{}
}

func startFileWatchers(
	offsetChans map[string]chan int64,
	converterChans map[string]timeConverterChan,
	content DirectoryContent,
	queues fileQueues,
	sortableRecordsChan chan<- sortableRecord,
	done chan<- struct{},
	patterns LogPatterns,
) error {
	actions := []func(){}

	for pattern, queue := range queues {
		// The last file in the queue is the current log file
		entry := queue[len(queue)-1]

		if path.Base(entry.filename) != pattern {
			return errorutil.Wrap(fmt.Errorf("Missing file: %s. Instead found: %s", pattern, entry.filename))
		}

		converterChan, ok := converterChans[pattern]

		if !ok {
			log.Fatal().Msgf("Failed to obtain offset chan for %s", pattern)
		}

		offsetChan, ok := offsetChans[pattern]

		if !ok {
			log.Fatal().Msgf("Failed to obtain offset chan for %s", pattern)
		}

		queueIndex := patterns.indexes[pattern]

		parsedRecordsChan := make(chan parsedRecord, maxNumberOfCachedElementsInTheHeap)

		actions = append(actions, func() {
			go startWatchingOnQueue(entry, queueIndex, offsetChan, content, parsedRecordsChan)
			go startTimestampingParsedLogs(converterChan, sortableRecordsChan, parsedRecordsChan, done)
		})
	}

	for _, f := range actions {
		f()
	}

	return nil
}

const (
	// While the importing of the archived logs has not finished,
	// how many new parsed logs do we keep in memory, received by
	// postfix in realtime?
	maxNumberOfCachedElementsInTheHeap = 2048
)

func publishNewLogsSorted(sortableRecordsChan <-chan sortableRecord, pub newLogsPublisher) <-chan struct{} {
	done := make(chan struct{})

	h := make(sortableRecordHeap, 0, maxNumberOfCachedElementsInTheHeap)

	heap.Init(&h)

	flushHeap := func() {
		for h.Len() > 0 {
			//nolint:forcetypeassert
			s := heap.Pop(&h).(sortableRecord)
			r := postfix.Record{Header: s.record.header, Payload: s.record.payload, Time: s.time, Location: s.record.loc}
			pub.Publish(r)
		}
	}

	go func() {
		// flushes the heap every two seconds
		ticker := time.NewTicker(2 * time.Second)
	loop:
		for {
			select {
			case r, ok := <-sortableRecordsChan:
				{
					if !ok {
						// channel has been closed
						break loop
					}

					heap.Push(&h, r)
					break
				}
			case <-ticker.C:
				flushHeap()
			}
		}

		flushHeap()

		close(pub.records)
		done <- struct{}{}
	}()

	return done
}

func filterNonEmptyQueues(queues fileQueues) fileQueues {
	r := fileQueues{}

	for pattern, queue := range queues {
		if len(queue) > 0 {
			r[pattern] = queue
		}
	}

	return r
}

func watchCurrentFilesForNewLogs(
	offsetChans map[string]chan int64,
	converterChans map[string]timeConverterChan,
	content DirectoryContent,
	queues fileQueues,
	pub newLogsPublisher,
	patterns LogPatterns,
) (waitForDone func(), cancelCall func(), returnError error) {
	nonEmptyQueues := filterNonEmptyQueues(queues)

	doneOnEveryWatcher := make(chan struct{}, len(nonEmptyQueues))

	// All watchers will write to this channel
	// and the publisher thread will read from it
	sortableRecordsChan := make(chan sortableRecord)

	if err := startFileWatchers(offsetChans, converterChans, content, nonEmptyQueues, sortableRecordsChan, doneOnEveryWatcher, patterns); err != nil {
		return func() {}, func() {}, errorutil.Wrap(err)
	}

	donePublishing := publishNewLogsSorted(sortableRecordsChan, pub)

	done := make(chan struct{})

	go func() {
		// wait until all watchers are finished
		for s := len(nonEmptyQueues); s > 0; s-- {
			<-doneOnEveryWatcher
		}

		close(sortableRecordsChan)

		<-donePublishing

		done <- struct{}{}
	}()

	cancel := make(chan struct{}, 1)

	waitForDone = func() {
		<-done
	}

	cancelCall = func() {
		cancel <- struct{}{}
	}

	return waitForDone, cancelCall, nil
}

func timeConverterChansFromQueues(queues fileQueues) map[string]timeConverterChan {
	chans := map[string]timeConverterChan{}

	for k := range queues {
		chans[k] = make(chan *parser.TimeConverter, 1)
	}

	return chans
}

func offsetChansFromQueues(queues fileQueues) map[string]chan int64 {
	chans := map[string]chan int64{}

	for k := range queues {
		chans[k] = make(chan int64, 1)
	}

	return chans
}

func (importer *DirectoryImporter) Run() error {
	return importer.run(true)
}

func (importer *DirectoryImporter) ImportOnly() error {
	return importer.run(false)
}

func (importer *DirectoryImporter) run(watch bool) error {
	queues, err := buildQueuesForDirImporter(importer.content, importer.patterns, importer.initialTime)

	if err != nil {
		return errorutil.Wrap(err)
	}

	newLogsPublisher := newLogsPublisher{records: make(chan postfix.Record)}

	converterChans := timeConverterChansFromQueues(queues)

	offsetChans := offsetChansFromQueues(queues)

	done, cancel, err := func() (func(), func(), error) {
		if watch {
			return watchCurrentFilesForNewLogs(offsetChans, converterChans, importer.content, queues, newLogsPublisher, importer.patterns)
		}

		return func() {}, func() {}, nil
	}()

	if err != nil {
		return errorutil.Wrap(err)
	}

	interruptWatching := func() {
		cancel()
		done()
	}

	err = importExistingLogs(offsetChans, converterChans, importer.content, queues, importer.pub, importer.initialTime, importer.announcer, importer.patterns)

	if err != nil {
		interruptWatching()
		return errorutil.Wrap(err)
	}

	if !watch {
		return nil
	}

	// Start really publishing the buffered records here, indefinitely
	for r := range newLogsPublisher.records {
		importer.pub.Publish(r)
	}

	// It should never get here in production, only used by the tests
	interruptWatching()

	return nil
}
