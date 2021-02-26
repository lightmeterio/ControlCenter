
//line cleanup.rl:1
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

// +build !codeanalysis

package rawparser


//line cleanup.rl:10

//line cleanup.gen.go:15
const cleanupMessageAccepted_start int = 1
const cleanupMessageAccepted_first_final int = 16
const cleanupMessageAccepted_error int = 0

const cleanupMessageAccepted_en_main int = 1


//line cleanup.rl:11

func parseCleanupMessageAccepted(data []byte) (CleanupMessageAccepted, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := CleanupMessageAccepted{}


//line cleanup.gen.go:34
	{
	cs = cleanupMessageAccepted_start
	}

//line cleanup.gen.go:39
	{
	if p == pe {
		goto _test_eof
	}
	switch cs {
	case 1:
		goto st_case_1
	case 0:
		goto st_case_0
	case 2:
		goto st_case_2
	case 3:
		goto st_case_3
	case 4:
		goto st_case_4
	case 5:
		goto st_case_5
	case 6:
		goto st_case_6
	case 7:
		goto st_case_7
	case 8:
		goto st_case_8
	case 9:
		goto st_case_9
	case 10:
		goto st_case_10
	case 11:
		goto st_case_11
	case 12:
		goto st_case_12
	case 13:
		goto st_case_13
	case 14:
		goto st_case_14
	case 16:
		goto st_case_16
	case 17:
		goto st_case_17
	case 15:
		goto st_case_15
	case 18:
		goto st_case_18
	case 19:
		goto st_case_19
	case 20:
		goto st_case_20
	case 21:
		goto st_case_21
	}
	goto st_out
	st_case_1:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto tr0
			}
		case data[p] > 70:
			if 97 <= data[p] && data[p] <= 102 {
				goto tr0
			}
		default:
			goto tr0
		}
		goto st0
st_case_0:
	st0:
		cs = 0
		goto _out
tr0:
//line common.rl:25
 tokBeg = p 
	goto st2
	st2:
		if p++; p == pe {
			goto _test_eof2
		}
	st_case_2:
//line cleanup.gen.go:118
		if data[p] == 58 {
			goto tr3
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st2
			}
		case data[p] > 70:
			if 97 <= data[p] && data[p] <= 102 {
				goto st2
			}
		default:
			goto st2
		}
		goto st0
tr3:
//line cleanup.rl:23

		r.Queue = data[tokBeg:p]
	
	goto st3
	st3:
		if p++; p == pe {
			goto _test_eof3
		}
	st_case_3:
//line cleanup.gen.go:146
		if data[p] == 32 {
			goto st4
		}
		goto st0
	st4:
		if p++; p == pe {
			goto _test_eof4
		}
	st_case_4:
		if data[p] == 109 {
			goto st5
		}
		goto st0
	st5:
		if p++; p == pe {
			goto _test_eof5
		}
	st_case_5:
		if data[p] == 101 {
			goto st6
		}
		goto st0
	st6:
		if p++; p == pe {
			goto _test_eof6
		}
	st_case_6:
		if data[p] == 115 {
			goto st7
		}
		goto st0
	st7:
		if p++; p == pe {
			goto _test_eof7
		}
	st_case_7:
		if data[p] == 115 {
			goto st8
		}
		goto st0
	st8:
		if p++; p == pe {
			goto _test_eof8
		}
	st_case_8:
		if data[p] == 97 {
			goto st9
		}
		goto st0
	st9:
		if p++; p == pe {
			goto _test_eof9
		}
	st_case_9:
		if data[p] == 103 {
			goto st10
		}
		goto st0
	st10:
		if p++; p == pe {
			goto _test_eof10
		}
	st_case_10:
		if data[p] == 101 {
			goto st11
		}
		goto st0
	st11:
		if p++; p == pe {
			goto _test_eof11
		}
	st_case_11:
		if data[p] == 45 {
			goto st12
		}
		goto st0
	st12:
		if p++; p == pe {
			goto _test_eof12
		}
	st_case_12:
		if data[p] == 105 {
			goto st13
		}
		goto st0
	st13:
		if p++; p == pe {
			goto _test_eof13
		}
	st_case_13:
		if data[p] == 100 {
			goto st14
		}
		goto st0
	st14:
		if p++; p == pe {
			goto _test_eof14
		}
	st_case_14:
		if data[p] == 61 {
			goto st16
		}
		goto st0
	st16:
		if p++; p == pe {
			goto _test_eof16
		}
	st_case_16:
		switch data[p] {
		case 60:
			goto st15
		case 62:
			goto st0
		}
		goto tr18
tr18:
//line common.rl:25
 tokBeg = p 
	goto st17
	st17:
		if p++; p == pe {
			goto _test_eof17
		}
	st_case_17:
//line cleanup.gen.go:271
		switch data[p] {
		case 60:
			goto st0
		case 62:
			goto st0
		}
		goto st17
	st15:
		if p++; p == pe {
			goto _test_eof15
		}
	st_case_15:
		if data[p] == 62 {
			goto st21
		}
		goto tr16
tr16:
//line common.rl:25
 tokBeg = p 
	goto st18
	st18:
		if p++; p == pe {
			goto _test_eof18
		}
	st_case_18:
//line cleanup.gen.go:297
		if data[p] == 62 {
			goto tr22
		}
		goto st18
tr22:
//line cleanup.rl:27

    r.MessageId = data[tokBeg:p]
  
	goto st19
	st19:
		if p++; p == pe {
			goto _test_eof19
		}
	st_case_19:
//line cleanup.gen.go:313
		if data[p] == 62 {
			goto st0
		}
		goto st20
	st20:
		if p++; p == pe {
			goto _test_eof20
		}
	st_case_20:
		if data[p] == 62 {
			goto st0
		}
		goto st20
	st21:
		if p++; p == pe {
			goto _test_eof21
		}
	st_case_21:
		goto st0
	st_out:
	_test_eof2: cs = 2; goto _test_eof
	_test_eof3: cs = 3; goto _test_eof
	_test_eof4: cs = 4; goto _test_eof
	_test_eof5: cs = 5; goto _test_eof
	_test_eof6: cs = 6; goto _test_eof
	_test_eof7: cs = 7; goto _test_eof
	_test_eof8: cs = 8; goto _test_eof
	_test_eof9: cs = 9; goto _test_eof
	_test_eof10: cs = 10; goto _test_eof
	_test_eof11: cs = 11; goto _test_eof
	_test_eof12: cs = 12; goto _test_eof
	_test_eof13: cs = 13; goto _test_eof
	_test_eof14: cs = 14; goto _test_eof
	_test_eof16: cs = 16; goto _test_eof
	_test_eof17: cs = 17; goto _test_eof
	_test_eof15: cs = 15; goto _test_eof
	_test_eof18: cs = 18; goto _test_eof
	_test_eof19: cs = 19; goto _test_eof
	_test_eof20: cs = 20; goto _test_eof
	_test_eof21: cs = 21; goto _test_eof

	_test_eof: {}
	if p == eof {
		switch cs {
		case 19, 21:
//line cleanup.rl:41

		return r, true
	
		case 17:
//line cleanup.rl:31

    r.MessageId = data[tokBeg:p]
  
//line cleanup.rl:41

		return r, true
	
		case 16, 20:
//line cleanup.rl:37

    r.Corrupted = true
  
//line cleanup.rl:41

		return r, true
	
		case 18:
//line cleanup.rl:27

    r.MessageId = data[tokBeg:p]
  
//line cleanup.rl:37

    r.Corrupted = true
  
//line cleanup.rl:41

		return r, true
	
//line cleanup.gen.go:394
		}
	}

	_out: {}
	}

//line cleanup.rl:47


	return r, false
}


//line cleanup.rl:53

//line cleanup.gen.go:410
const cleanupMilterReject_start int = 1
const cleanupMilterReject_first_final int = 20
const cleanupMilterReject_error int = 0

const cleanupMilterReject_en_main int = 1


//line cleanup.rl:54

func parseCleanupMilterReject(data []byte) (CleanupMilterReject, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := CleanupMilterReject{}


//line cleanup.gen.go:429
	{
	cs = cleanupMilterReject_start
	}

//line cleanup.gen.go:434
	{
	if p == pe {
		goto _test_eof
	}
	switch cs {
	case 1:
		goto st_case_1
	case 0:
		goto st_case_0
	case 2:
		goto st_case_2
	case 3:
		goto st_case_3
	case 4:
		goto st_case_4
	case 5:
		goto st_case_5
	case 6:
		goto st_case_6
	case 7:
		goto st_case_7
	case 8:
		goto st_case_8
	case 9:
		goto st_case_9
	case 10:
		goto st_case_10
	case 11:
		goto st_case_11
	case 12:
		goto st_case_12
	case 13:
		goto st_case_13
	case 14:
		goto st_case_14
	case 15:
		goto st_case_15
	case 16:
		goto st_case_16
	case 17:
		goto st_case_17
	case 18:
		goto st_case_18
	case 19:
		goto st_case_19
	case 20:
		goto st_case_20
	}
	goto st_out
	st_case_1:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto tr0
			}
		case data[p] > 70:
			if 97 <= data[p] && data[p] <= 102 {
				goto tr0
			}
		default:
			goto tr0
		}
		goto st0
st_case_0:
	st0:
		cs = 0
		goto _out
tr0:
//line common.rl:25
 tokBeg = p 
	goto st2
	st2:
		if p++; p == pe {
			goto _test_eof2
		}
	st_case_2:
//line cleanup.gen.go:511
		if data[p] == 58 {
			goto tr3
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st2
			}
		case data[p] > 70:
			if 97 <= data[p] && data[p] <= 102 {
				goto st2
			}
		default:
			goto st2
		}
		goto st0
tr3:
//line cleanup.rl:66

		r.Queue = data[tokBeg:p]
	
	goto st3
	st3:
		if p++; p == pe {
			goto _test_eof3
		}
	st_case_3:
//line cleanup.gen.go:539
		if data[p] == 32 {
			goto st4
		}
		goto st0
	st4:
		if p++; p == pe {
			goto _test_eof4
		}
	st_case_4:
		if data[p] == 109 {
			goto st5
		}
		goto st0
	st5:
		if p++; p == pe {
			goto _test_eof5
		}
	st_case_5:
		if data[p] == 105 {
			goto st6
		}
		goto st0
	st6:
		if p++; p == pe {
			goto _test_eof6
		}
	st_case_6:
		if data[p] == 108 {
			goto st7
		}
		goto st0
	st7:
		if p++; p == pe {
			goto _test_eof7
		}
	st_case_7:
		if data[p] == 116 {
			goto st8
		}
		goto st0
	st8:
		if p++; p == pe {
			goto _test_eof8
		}
	st_case_8:
		if data[p] == 101 {
			goto st9
		}
		goto st0
	st9:
		if p++; p == pe {
			goto _test_eof9
		}
	st_case_9:
		if data[p] == 114 {
			goto st10
		}
		goto st0
	st10:
		if p++; p == pe {
			goto _test_eof10
		}
	st_case_10:
		if data[p] == 45 {
			goto st11
		}
		goto st0
	st11:
		if p++; p == pe {
			goto _test_eof11
		}
	st_case_11:
		if data[p] == 114 {
			goto st12
		}
		goto st0
	st12:
		if p++; p == pe {
			goto _test_eof12
		}
	st_case_12:
		if data[p] == 101 {
			goto st13
		}
		goto st0
	st13:
		if p++; p == pe {
			goto _test_eof13
		}
	st_case_13:
		if data[p] == 106 {
			goto st14
		}
		goto st0
	st14:
		if p++; p == pe {
			goto _test_eof14
		}
	st_case_14:
		if data[p] == 101 {
			goto st15
		}
		goto st0
	st15:
		if p++; p == pe {
			goto _test_eof15
		}
	st_case_15:
		if data[p] == 99 {
			goto st16
		}
		goto st0
	st16:
		if p++; p == pe {
			goto _test_eof16
		}
	st_case_16:
		if data[p] == 116 {
			goto st17
		}
		goto st0
	st17:
		if p++; p == pe {
			goto _test_eof17
		}
	st_case_17:
		if data[p] == 58 {
			goto st18
		}
		goto st0
	st18:
		if p++; p == pe {
			goto _test_eof18
		}
	st_case_18:
		if data[p] == 32 {
			goto st19
		}
		goto st0
	st19:
		if p++; p == pe {
			goto _test_eof19
		}
	st_case_19:
		goto tr20
tr20:
//line common.rl:25
 tokBeg = p 
//line cleanup.rl:72

		r.ExtraMessage = data[tokBeg:eof]
    return r, true
  
	goto st20
tr21:
//line cleanup.rl:72

		r.ExtraMessage = data[tokBeg:eof]
    return r, true
  
	goto st20
	st20:
		if p++; p == pe {
			goto _test_eof20
		}
	st_case_20:
//line cleanup.gen.go:706
		goto tr21
	st_out:
	_test_eof2: cs = 2; goto _test_eof
	_test_eof3: cs = 3; goto _test_eof
	_test_eof4: cs = 4; goto _test_eof
	_test_eof5: cs = 5; goto _test_eof
	_test_eof6: cs = 6; goto _test_eof
	_test_eof7: cs = 7; goto _test_eof
	_test_eof8: cs = 8; goto _test_eof
	_test_eof9: cs = 9; goto _test_eof
	_test_eof10: cs = 10; goto _test_eof
	_test_eof11: cs = 11; goto _test_eof
	_test_eof12: cs = 12; goto _test_eof
	_test_eof13: cs = 13; goto _test_eof
	_test_eof14: cs = 14; goto _test_eof
	_test_eof15: cs = 15; goto _test_eof
	_test_eof16: cs = 16; goto _test_eof
	_test_eof17: cs = 17; goto _test_eof
	_test_eof18: cs = 18; goto _test_eof
	_test_eof19: cs = 19; goto _test_eof
	_test_eof20: cs = 20; goto _test_eof

	_test_eof: {}
	_out: {}
	}

//line cleanup.rl:79


	return r, false
}
