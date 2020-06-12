package dirwatcher

import (
	"bufio"
	"container/heap"
	"errors"
	"io"
	"log"
	"path"
	"regexp"
	"sort"
	"strconv"
	"time"

	"gitlab.com/lightmeter/controlcenter/data"
	"gitlab.com/lightmeter/controlcenter/data/postfix"
	"gitlab.com/lightmeter/controlcenter/util"
	parser "gitlab.com/lightmeter/postfix-log-parser"
)

type fileEntry struct {
	filename         string
	modificationTime time.Time
}

type fileEntryList []fileEntry

type logPatterns []string

type fileQueues map[string]fileEntryList

func sortedEntriesFilteredByPatternAndMoreRecentThanTime(list fileEntryList, pattern string, initialTime time.Time) fileEntryList {
	// NOTE: we are using the default logrotate naming convension. More info at:
	// http://man7.org/linux/man-pages/man5/logrotate.conf.5.html
	reg, err := regexp.Compile(`^(` + pattern + `)(\.(\d+)(\.gz)?)?$`)

	util.MustSucceed(err, `trying to build regexp for pattern "`+pattern+`"`)

	entries := make(fileEntryList, 0, len(list))

	type rec struct {
		entry fileEntry
		index int
	}

	recs := []rec{}

	for _, entry := range list {
		basename := path.Base(entry.filename)
		matches := reg.FindSubmatch([]byte(basename))

		if len(matches) == 0 || entry.modificationTime.Before(initialTime) {
			continue
		}

		index := func() int {
			if len(matches[3]) == 0 {
				return 0
			}

			index, err := strconv.Atoi(string(matches[3]))
			util.MustSucceed(err, "Atoi")

			return index
		}()

		recs = append(recs, rec{entry: entry, index: index})
	}

	sort.Slice(recs, func(i, j int) bool {
		// desc sort, so we have mail.log.2.gz, mail.log.1, mail.log
		return recs[i].index > recs[j].index
	})

	for _, r := range recs {
		entries = append(entries, r.entry)
	}

	return entries
}

func buildFilesToImport(list fileEntryList, patterns logPatterns, initialTime time.Time) fileQueues {
	if len(patterns) == 0 {
		return fileQueues{}
	}

	queues := make(fileQueues, len(patterns))

	for _, pattern := range patterns {
		queues[pattern] = sortedEntriesFilteredByPatternAndMoreRecentThanTime(list, pattern, initialTime)
	}

	return queues
}

// Given a leap year, what nth second is a time on it?
func secondInTheYear(v parser.Time) float64 {
	asRefTime := func(v parser.Time) time.Time {
		return time.Date(2000, v.Month, int(v.Day), int(v.Hour), int(v.Minute), int(v.Second), 0, time.UTC)
	}

	return asRefTime(v).Sub(
		asRefTime(parser.Time{Month: time.January, Day: 1, Hour: 0, Minute: 0, Second: 0})).Seconds()
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
		return util.WrapError(err)
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
		return util.WrapError(err)
	}()
}

func guessInitialDateForFile(reader io.Reader, modificationTime time.Time) (time.Time, error) {
	scanner := bufio.NewScanner(reader)

	timeFirstLine, ok, err := readFirstLine(scanner)

	if !ok {
		// empty file
		return modificationTime, nil
	}

	if err != nil && !parser.IsRecoverableError(errors.Unwrap(err)) {
		// failed to read first line
		return time.Time{}, util.WrapError(err)
	}

	secondsInYearFirstLine := secondInTheYear(timeFirstLine)

	timeLastLine, ok, err := readLastLine(scanner)

	computeYear := func(a, b float64) int {
		yearOffset := 0

		if a < b {
			yearOffset++
		}

		return modificationTime.Year() - yearOffset
	}

	secondsInYearModificationTime := secondInTheYear(parser.Time{
		Month:  modificationTime.Month(),
		Day:    uint8(modificationTime.Day()),
		Hour:   uint8(modificationTime.Hour()),
		Minute: uint8(modificationTime.Minute()),
		Second: uint8(modificationTime.Second()),
	})

	if !ok {
		// one line file
		year := computeYear(secondsInYearModificationTime, secondsInYearFirstLine)
		return timeFirstLine.Time(year, modificationTime.Location()), nil
	}

	if err != nil && !parser.IsRecoverableError(errors.Unwrap(err)) {
		// failed reading last line
		return time.Time{}, util.WrapError(err)
	}

	secondsInYearLastLine := secondInTheYear(timeLastLine)

	ordered := func(a, b, c float64) bool {
		return a <= b && b <= c
	}

	offset := func(B, E, M float64) int {
		// B = Begin
		// E = End
		// M = Modified

		// NOTE: This code can be simplified, but enumerating all possible combinations
		// makes it clear we are not missing any case
		switch {
		case ordered(B, E, M):
			return 0
		case ordered(B, M, E):
			return 1
		case ordered(E, B, M):
			return 1
		case ordered(E, M, B):
			return 1
		case ordered(M, B, E):
			return 1
		case ordered(M, E, B):
			return 1
		default:
			panic("SPANK SPANK! This should not be possible, but it turns out it is")
		}
	}

	year := modificationTime.Year() - offset(secondsInYearFirstLine, secondsInYearLastLine, secondsInYearModificationTime)

	return timeFirstLine.Time(year, modificationTime.Location()), nil
}

type fileDescriptor struct {
	modificationTime time.Time
	reader           fileReader
}

var ErrEmptyFileList = errors.New(`Empty list!`)

func findEarlierstTimeFromFiles(files []fileDescriptor) (time.Time, error) {
	if len(files) == 0 {
		return time.Time{}, util.WrapError(ErrEmptyFileList)
	}

	var t time.Time

	// NOTE: this code does not work for files from before the Unix epoch
	for _, file := range files {
		ft, err := guessInitialDateForFile(file.reader, file.modificationTime)

		if err != nil {
			return time.Time{}, util.WrapError(err)
		}

		if (ft.Before(t) || t == time.Time{}) {
			t = ft
		}
	}

	return t, nil
}

func FindInitialLogTime(content DirectoryContent) (time.Time, error) {
	queues, err := buildQueuesForDirImporter(content, patterns, time.Time{})

	if err != nil {
		return time.Time{}, util.WrapError(err)
	}

	descriptors := []fileDescriptor{}

	fileClosers := []func(){}

	defer func() {
		for _, f := range fileClosers {
			f()
		}
	}()

	for _, queue := range queues {
		if len(queue) == 0 {
			continue
		}

		entry := queue[0]

		filename := entry.filename

		reader, err := content.readerForEntry(filename)

		if err != nil {
			return time.Time{}, util.WrapError(err)
		}

		fileClosers = append(fileClosers, func() {
			util.MustSucceed(reader.Close(), "Closing file: "+filename)
		})

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
	fileEntries() fileEntryList
	modificationTimeForEntry(filename string) (time.Time, error)
	readerForEntry(filename string) (fileReader, error)
	watcherForEntry(filename string, offset int64) (fileWatcher, error)
	readSeekerForEntry(filename string) (fileReadSeeker, error)
}

type DirectoryImporter struct {
	content     DirectoryContent
	pub         data.Publisher
	initialTime time.Time
}

func NewDirectoryImporter(
	content DirectoryContent,
	pub data.Publisher,
	initialTime time.Time,
) DirectoryImporter {
	return DirectoryImporter{content, pub, initialTime}
}

type timedRecord struct {
	header  parser.Header
	payload parser.Payload
	time    time.Time
}

func readFromReader(reader io.Reader,
	filename string,
	onNewRecord func(parser.Header, parser.Payload)) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Bytes()

		h, p, err := parser.Parse(line)

		if parser.IsRecoverableError(err) {
			onNewRecord(h, p)
		}
	}
}

var ErrEmptyDirectory = errors.New("Empty Directory")

func buildQueuesForDirImporter(content DirectoryContent, patterns logPatterns, initialTime time.Time) (fileQueues, error) {
	entries := content.fileEntries()

	if len(entries) == 0 {
		return fileQueues{}, util.WrapError(ErrEmptyDirectory)
	}

	queues := buildFilesToImport(entries, patterns, initialTime)

	if len(queues) == 0 {
		return fileQueues{}, util.WrapError(ErrEmptyDirectory)
	}

	for _, q := range queues {
		if len(q) != 0 {
			return queues, nil
		}
	}

	return fileQueues{}, util.WrapError(ErrEmptyDirectory)
}

var (
	patterns       = logPatterns{"mail.log", "mail.err", "mail.warn"}
	patternIndexes = map[string]int{}
)

func init() {
	for i, v := range patterns {
		patternIndexes[v] = i
	}
}

type timeConverterChan chan *postfix.TimeConverter

type queueProcessor struct {
	readers       []fileReader
	scanners      []*bufio.Scanner
	entries       fileEntryList
	record        timedRecord
	currentIndex  int
	converter     *postfix.TimeConverter
	converterChan timeConverterChan
	pattern       string
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
		return nil, 0, util.WrapError(err)
	}

	offset, err := readSeeker.Seek(0, io.SeekEnd)

	defer func() {
		if err != nil {
			util.MustSucceed(readSeeker.Close(), "Closing on seeking file to end")
		}
	}()

	if err != nil {
		return nil, 0, util.WrapError(err)
	}

	_, err = readSeeker.Seek(0, io.SeekStart)

	if err != nil {
		return nil, 0, util.WrapError(err)
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
			return nil, nil, util.WrapError(err)
		}

		// inform the "watch current log file" thread about the offset
		// in the file it should start watching from
		offsetChan <- offset

		return reader, bufio.NewScanner(reader), nil
	}

	reader, err := content.readerForEntry(entry.filename)

	if err != nil {
		return nil, nil, util.WrapError(err)
	}

	return reader, bufio.NewScanner(reader), nil
}

func processorForQueue(offsetChan chan int64, converterChan timeConverterChan, content DirectoryContent, pattern string, entries fileEntryList) (queueProcessor, error) {
	readers := []fileReader{}
	scanners := []*bufio.Scanner{}

	for _, entry := range entries {
		reader, scanner, err := buildReaderAndScannerForEntry(offsetChan, content, pattern, entry)

		if err != nil {
			return queueProcessor{}, util.WrapError(err)
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

func buildQueueProcessors(offsetChans map[string]chan int64, converterChans map[string]timeConverterChan, content DirectoryContent, queues fileQueues) ([]*queueProcessor, error) {
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
			return []*queueProcessor{}, util.WrapError(err)
		}

		index := patternIndexes[k]

		p[index] = &processor
	}

	return p, nil
}

func createConverterForQueueProcessor(p *queueProcessor, content DirectoryContent, header parser.Header) (*postfix.TimeConverter, error) {
	modificationTime, err := content.modificationTimeForEntry(p.entries[p.currentIndex].filename)

	if err != nil {
		return nil, util.WrapError(err)
	}

	reader, err := content.readerForEntry(p.entries[p.currentIndex].filename)

	if err != nil {
		return nil, util.WrapError(err)
	}

	defer func() {
		util.MustSucceed(reader.Close(), "Closing first file in queue")
	}()

	initialTime, err := guessInitialDateForFile(reader, modificationTime)

	if err != nil {
		return nil, util.WrapError(err)
	}

	// Copy the pattern string so it can be moved into the log lambda below
	// NOTE: I really miss explicit ownership checked at compile time :-(
	pattern := p.pattern

	converter := postfix.NewTimeConverter(
		header.Time,
		initialTime.Year(),
		initialTime.Location(),
		func(int, parser.Time, parser.Time) {
			log.Println("Changed Year on log queue", pattern)
		})

	// workaround, make converter escape to the heap
	return &converter, nil
}

func updateQueueProcessor(p *queueProcessor, content DirectoryContent) (bool, error) {
	// tries to read something from the queue, ignoring it on the next iteration
	// if nothing is left to be read
	for {
		thereAreFilesToBeProcessed := p.currentIndex < len(p.readers)

		if !thereAreFilesToBeProcessed {
			// ended processing queue, but moves the time converter to the file watcher to be reused there
			p.converterChan <- p.converter
			return false, nil
		}

		scanner := p.scanners[p.currentIndex]

		if !scanner.Scan() {
			// file ended, use next one
			if err := p.readers[p.currentIndex].Close(); err != nil {
				return false, util.WrapError(err)
			}

			log.Println("Finished importing log file:", p.entries[p.currentIndex].filename)

			p.currentIndex++

			// moves to the next file in the queue
			continue
		}

		// Successfully read
		header, payload, err := parser.Parse(scanner.Bytes())

		if !parser.IsRecoverableError(err) {
			return false, util.WrapError(err)
		}

		if p.converter == nil {
			converter, err := createConverterForQueueProcessor(p, content, header)

			if err != nil {
				return false, util.WrapError(err)
			}

			p.converter = converter
		}

		convertedTime := p.converter.Convert(header.Time)

		p.record = timedRecord{header: header, payload: payload, time: convertedTime}

		return true, nil
	}
}

func updateQueueProcessors(content DirectoryContent, processors []*queueProcessor, toBeUpdated int) ([]*queueProcessor, error) {
	updatedProcessors := make([]*queueProcessor, 0, len(processors))

	for i, p := range processors {
		isFirstExecution := toBeUpdated != -1

		if isFirstExecution && i != toBeUpdated {
			updatedProcessors = append(updatedProcessors, p)
			continue
		}

		shouldKeepProcessor, err := updateQueueProcessor(p, content)

		if err != nil {
			return []*queueProcessor{}, util.WrapError(err)
		}

		if shouldKeepProcessor {
			updatedProcessors = append(updatedProcessors, p)
		}
	}

	return updatedProcessors, nil
}

func chooseIndexForOldestElement(queueProcessors []*queueProcessor) int {
	chosenIndex := -1

	for i, p := range queueProcessors {
		if chosenIndex == -1 || queueProcessors[chosenIndex].record.time.After(p.record.time) {
			chosenIndex = i
		}
	}

	if chosenIndex == -1 {
		panic("BUG: your algorithm sucks!")
	}

	return chosenIndex
}

func importExistingLogs(
	offsetChans map[string]chan int64,
	converterChans map[string]timeConverterChan,
	content DirectoryContent,
	queues fileQueues,
	pub data.Publisher,
	initialTime time.Time,
) error {
	/*
	 * Open all log files, including archived (compressed or not, but logrotate)
	 * and read them line by line, publishing them in the right order they were generated (or
	 * close enough, as the lines have only precision of second, so it's not a "stable sort"),
	 * so the order among different lines on the same second is not deterministic.
	 */

	initialImportTime := time.Now()

	queueProcessors, err := buildQueueProcessors(offsetChans, converterChans, content, queues)

	if err != nil {
		return util.WrapError(err)
	}

	toBeUpdated := -1

	for {
		updatedQueueProcessors, err := updateQueueProcessors(content, queueProcessors, toBeUpdated)

		if err != nil {
			return util.WrapError(err)
		}

		queueProcessors = updatedQueueProcessors

		if len(queueProcessors) == 0 {
			elapsedTime := time.Since(initialImportTime)
			log.Println("Finished importing postfix log directory in:", elapsedTime)
			return nil
		}

		toBeUpdated = chooseIndexForOldestElement(queueProcessors)

		t := queueProcessors[toBeUpdated].record

		if t.time.After(initialTime) {
			pub.Publish(data.Record{Header: t.header, Payload: t.payload})
		}
	}
}

type newLogsPublisher struct {
	// a temporary buffer for the new lines that arrive before the archived logs are imported
	// so we publish them in chronological order
	records chan timedRecord
}

func (pub newLogsPublisher) Publish(r timedRecord) {
	pub.records <- r
}

func (pub newLogsPublisher) Close() {
	close(pub.records)
}

type sortableRecord struct {
	record parsedRecord
	time   time.Time
}

func (r sortableRecord) Less(other sortableRecord) bool {
	// Compare lexicographically
	// NOTE: I wish go had something like C++ std::tuple, which would simplify
	// this to one line:
	// `return make_tuple(r.timedRecord, r.queueIndex, r.sequence) < make_tuple(other.timedRecord, other.queueIndex, other.sequence)`

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

	util.MustSucceed(err, "File watcher for: "+entry.filename)

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
// wait until the initial import is finished, obtaining the time converter from it
func startTimestampingParsedLogs(
	converterChan timeConverterChan,
	sortableRecordsChan chan<- sortableRecord,
	parsedRecordsChan <-chan parsedRecord,
	done chan<- struct{}) {

	// While the initial import happens,
	// the time converter is not available,
	// being owned by the import process.
	// once it's finished, we take ownership
	// over the converter and start using it from
	// the exact point in time the import stopped.
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
	done chan<- struct{}) {

	actions := []func(){}

	for pattern, queue := range queues {
		// The last file in the queue is the current log file
		entry := queue[len(queue)-1]

		if path.Base(entry.filename) != pattern {
			log.Fatalln("Missing file", pattern, ". Instead, found: ", entry.filename)
		}

		converterChan, ok := converterChans[pattern]

		if !ok {
			log.Fatalln("Failed to obtain offset chan for", pattern)
		}

		offsetChan, ok := offsetChans[pattern]

		if !ok {
			log.Fatalln("Failed to obtain offset chan for", pattern)
		}

		queueIndex := patternIndexes[pattern]

		parsedRecordsChan := make(chan parsedRecord, maxNumberOfCachedElementsInTheHeap)

		actions = append(actions, func() {
			go startWatchingOnQueue(entry, queueIndex, offsetChan, content, parsedRecordsChan)
			go startTimestampingParsedLogs(converterChan, sortableRecordsChan, parsedRecordsChan, done)
		})
	}

	for _, f := range actions {
		f()
	}
}

const (
	// While the importing of the archived logs has not finished,
	// how many new parsed logs do we keep in memory, received by
	// postfix in realtime?
	maxNumberOfCachedElementsInTheHeap = 500000
)

func publishNewLogsSorted(sortableRecordsChan <-chan sortableRecord, pub newLogsPublisher) <-chan struct{} {
	done := make(chan struct{})

	h := make(sortableRecordHeap, 0, maxNumberOfCachedElementsInTheHeap)

	heap.Init(&h)

	flushHeap := func() {
		for h.Len() > 0 {
			s := heap.Pop(&h).(sortableRecord)
			r := timedRecord{header: s.record.header, payload: s.record.payload, time: s.time}
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

		pub.Close()
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
	pub newLogsPublisher) (waitForDone func(), cancelCall func()) {

	nonEmptyQueues := filterNonEmptyQueues(queues)

	doneOnEveryWatcher := make(chan struct{}, len(nonEmptyQueues))

	// All watchers will write to this channel
	// and the publisher thread will read from it
	sortableRecordsChan := make(chan sortableRecord)

	startFileWatchers(offsetChans, converterChans, content, nonEmptyQueues, sortableRecordsChan, doneOnEveryWatcher)

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

	return waitForDone, cancelCall
}

func timeConverterChansFromQueues(queues fileQueues) map[string]timeConverterChan {
	chans := map[string]timeConverterChan{}

	for k := range queues {
		chans[k] = make(chan *postfix.TimeConverter, 1)
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
	queues, err := buildQueuesForDirImporter(importer.content, patterns, importer.initialTime)

	if err != nil {
		return util.WrapError(err)
	}

	newLogsPublisher := newLogsPublisher{records: make(chan timedRecord)}

	converterChans := timeConverterChansFromQueues(queues)

	offsetChans := offsetChansFromQueues(queues)

	done, cancel := func() (func(), func()) {
		if watch {
			return watchCurrentFilesForNewLogs(offsetChans, converterChans, importer.content, queues, newLogsPublisher)
		}

		return func() {}, func() {}
	}()

	interruptWatching := func() {
		cancel()
		done()
	}

	if err := importExistingLogs(offsetChans, converterChans, importer.content, queues, importer.pub, importer.initialTime); err != nil {
		interruptWatching()
		return util.WrapError(err)
	}

	if !watch {
		return nil
	}

	// Start really publishing the buffered records here, indefinitely
	for r := range newLogsPublisher.records {
		// TODO: we are losing the converted time here, that can be useful for the publisher
		// Maybe we should include it as well, to avoid it to be recalculated further in the data flow
		importer.pub.Publish(data.Record{Header: r.header, Payload: r.payload})
	}

	// It should never get here in production, only used by the tests
	interruptWatching()

	return nil
}
