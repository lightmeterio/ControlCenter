
//line cleanup.rl:1
// +build !codeanalysis

package rawparser


//line cleanup.rl:6

//line cleanup.gen.go:11
const cleanupMessageAccepted_start int = 1
const cleanupMessageAccepted_first_final int = 18
const cleanupMessageAccepted_error int = 0

const cleanupMessageAccepted_en_main int = 1


//line cleanup.rl:7

func parseCleanupMessageAccepted(data []byte) (CleanupMessageAccepted, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := CleanupMessageAccepted{}


//line cleanup.gen.go:30
	{
	cs = cleanupMessageAccepted_start
	}

//line cleanup.gen.go:35
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
	case 18:
		goto st_case_18
	case 16:
		goto st_case_16
	case 17:
		goto st_case_17
	case 19:
		goto st_case_19
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
//line common.rl:19
 tokBeg = p 
	goto st2
	st2:
		if p++; p == pe {
			goto _test_eof2
		}
	st_case_2:
//line cleanup.gen.go:110
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
//line cleanup.rl:19

		r.Queue = data[tokBeg:p]
	
	goto st3
	st3:
		if p++; p == pe {
			goto _test_eof3
		}
	st_case_3:
//line cleanup.gen.go:138
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
			goto st15
		}
		goto st0
	st15:
		if p++; p == pe {
			goto _test_eof15
		}
	st_case_15:
		switch data[p] {
		case 32:
			goto st0
		case 39:
			goto st0
		case 60:
			goto st16
		case 62:
			goto st0
		}
		goto tr16
tr16:
//line common.rl:19
 tokBeg = p 
//line cleanup.rl:27

    r.MessageId = data[tokBeg:eof]
  
//line cleanup.rl:31

		return r, true
	
	goto st18
tr21:
//line cleanup.rl:27

    r.MessageId = data[tokBeg:eof]
  
//line cleanup.rl:31

		return r, true
	
	goto st18
	st18:
		if p++; p == pe {
			goto _test_eof18
		}
	st_case_18:
//line cleanup.gen.go:285
		switch data[p] {
		case 32:
			goto st0
		case 39:
			goto st0
		case 60:
			goto st0
		case 62:
			goto st0
		}
		goto tr21
	st16:
		if p++; p == pe {
			goto _test_eof16
		}
	st_case_16:
		if data[p] == 62 {
			goto st0
		}
		goto tr18
tr18:
//line common.rl:19
 tokBeg = p 
	goto st17
	st17:
		if p++; p == pe {
			goto _test_eof17
		}
	st_case_17:
//line cleanup.gen.go:315
		if data[p] == 62 {
			goto tr20
		}
		goto st17
tr20:
//line cleanup.rl:23

    r.MessageId = data[tokBeg:p]
  
//line cleanup.rl:31

		return r, true
	
	goto st19
	st19:
		if p++; p == pe {
			goto _test_eof19
		}
	st_case_19:
//line cleanup.gen.go:335
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
	_test_eof15: cs = 15; goto _test_eof
	_test_eof18: cs = 18; goto _test_eof
	_test_eof16: cs = 16; goto _test_eof
	_test_eof17: cs = 17; goto _test_eof
	_test_eof19: cs = 19; goto _test_eof

	_test_eof: {}
	_out: {}
	}

//line cleanup.rl:37


	return r, false
}
