
//line header.rl:1
// +build !codeanalysis

package rawparser


//line header.rl:6

//line header.gen.go:11
const headerPostfixPart_start int = 1
const headerPostfixPart_first_final int = 19
const headerPostfixPart_error int = 0

const headerPostfixPart_en_main int = 1


//line header.rl:7

func parseHeaderPostfixPart(h *RawHeader, data []byte) (int, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof


//line header.gen.go:28
	{
	cs = headerPostfixPart_start
	}

//line header.gen.go:33
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
	case 19:
		goto st_case_19
	case 10:
		goto st_case_10
	case 11:
		goto st_case_11
	case 12:
		goto st_case_12
	case 13:
		goto st_case_13
	case 20:
		goto st_case_20
	case 14:
		goto st_case_14
	case 21:
		goto st_case_21
	case 15:
		goto st_case_15
	case 16:
		goto st_case_16
	case 17:
		goto st_case_17
	case 18:
		goto st_case_18
	}
	goto st_out
	st_case_1:
		if data[p] == 46 {
			goto tr0
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto tr0
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
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
//line header.rl:15
 tokBeg = p 
	goto st2
	st2:
		if p++; p == pe {
			goto _test_eof2
		}
	st_case_2:
//line header.gen.go:115
		switch data[p] {
		case 32:
			goto tr2
		case 46:
			goto st2
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st2
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st2
			}
		default:
			goto st2
		}
		goto st0
tr2:
//line header.rl:17

		h.Host = data[tokBeg:p]
	
	goto st3
	st3:
		if p++; p == pe {
			goto _test_eof3
		}
	st_case_3:
//line header.gen.go:146
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto tr4
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto tr4
			}
		default:
			goto tr4
		}
		goto st0
tr4:
//line header.rl:15
 tokBeg = p 
	goto st4
	st4:
		if p++; p == pe {
			goto _test_eof4
		}
	st_case_4:
//line header.gen.go:169
		switch data[p] {
		case 45:
			goto tr5
		case 47:
			goto tr6
		case 58:
			goto tr8
		case 91:
			goto tr9
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st4
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st4
			}
		default:
			goto st4
		}
		goto st0
tr5:
//line header.rl:21

		h.Process = data[tokBeg:p]
	
	goto st5
	st5:
		if p++; p == pe {
			goto _test_eof5
		}
	st_case_5:
//line header.gen.go:204
		if data[p] == 47 {
			goto st0
		}
		goto tr10
tr10:
//line header.rl:15
 tokBeg = p 
	goto st6
tr28:
//line header.rl:33

		h.ProcessID = data[tokBeg:p]
	
	goto st6
	st6:
		if p++; p == pe {
			goto _test_eof6
		}
	st_case_6:
//line header.gen.go:224
		switch data[p] {
		case 47:
			goto tr12
		case 58:
			goto tr13
		case 91:
			goto tr14
		}
		goto st6
tr6:
//line header.rl:21

		h.Process = data[tokBeg:p]
	
	goto st7
tr12:
//line header.rl:25

		h.ProcessIP = data[tokBeg:p]
	
	goto st7
	st7:
		if p++; p == pe {
			goto _test_eof7
		}
	st_case_7:
//line header.gen.go:251
		if data[p] == 93 {
			goto st0
		}
		goto tr15
tr15:
//line header.rl:15
 tokBeg = p 
	goto st8
	st8:
		if p++; p == pe {
			goto _test_eof8
		}
	st_case_8:
//line header.gen.go:265
		switch data[p] {
		case 58:
			goto tr17
		case 91:
			goto tr18
		case 93:
			goto st0
		}
		goto st8
tr17:
//line header.rl:29

		h.Daemon = data[tokBeg:p]
	
	goto st9
	st9:
		if p++; p == pe {
			goto _test_eof9
		}
	st_case_9:
//line header.gen.go:286
		switch data[p] {
		case 32:
			goto tr19
		case 58:
			goto tr17
		case 91:
			goto tr18
		case 93:
			goto st0
		}
		goto st8
tr19:
//line header.rl:37

		return p, true
	
	goto st19
	st19:
		if p++; p == pe {
			goto _test_eof19
		}
	st_case_19:
//line header.gen.go:309
		switch data[p] {
		case 58:
			goto tr17
		case 91:
			goto tr18
		case 93:
			goto st0
		}
		goto st8
tr18:
//line header.rl:29

		h.Daemon = data[tokBeg:p]
	
	goto st10
	st10:
		if p++; p == pe {
			goto _test_eof10
		}
	st_case_10:
//line header.gen.go:330
		switch data[p] {
		case 58:
			goto tr17
		case 91:
			goto tr18
		case 93:
			goto st0
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto tr20
		}
		goto st8
tr20:
//line header.rl:15
 tokBeg = p 
	goto st11
	st11:
		if p++; p == pe {
			goto _test_eof11
		}
	st_case_11:
//line header.gen.go:352
		switch data[p] {
		case 58:
			goto tr17
		case 91:
			goto tr18
		case 93:
			goto tr22
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st11
		}
		goto st8
tr22:
//line header.rl:33

		h.ProcessID = data[tokBeg:p]
	
	goto st12
	st12:
		if p++; p == pe {
			goto _test_eof12
		}
	st_case_12:
//line header.gen.go:376
		if data[p] == 58 {
			goto st13
		}
		goto st0
tr8:
//line header.rl:21

		h.Process = data[tokBeg:p]
	
	goto st13
	st13:
		if p++; p == pe {
			goto _test_eof13
		}
	st_case_13:
//line header.gen.go:392
		if data[p] == 32 {
			goto tr24
		}
		goto st0
tr24:
//line header.rl:37

		return p, true
	
	goto st20
	st20:
		if p++; p == pe {
			goto _test_eof20
		}
	st_case_20:
//line header.gen.go:408
		goto st0
tr13:
//line header.rl:25

		h.ProcessIP = data[tokBeg:p]
	
	goto st14
	st14:
		if p++; p == pe {
			goto _test_eof14
		}
	st_case_14:
//line header.gen.go:421
		switch data[p] {
		case 32:
			goto tr25
		case 47:
			goto tr12
		case 58:
			goto tr13
		case 91:
			goto tr14
		}
		goto st6
tr25:
//line header.rl:37

		return p, true
	
	goto st21
	st21:
		if p++; p == pe {
			goto _test_eof21
		}
	st_case_21:
//line header.gen.go:444
		switch data[p] {
		case 47:
			goto tr12
		case 58:
			goto tr13
		case 91:
			goto tr14
		}
		goto st6
tr14:
//line header.rl:25

		h.ProcessIP = data[tokBeg:p]
	
	goto st15
	st15:
		if p++; p == pe {
			goto _test_eof15
		}
	st_case_15:
//line header.gen.go:465
		switch data[p] {
		case 47:
			goto tr12
		case 58:
			goto tr13
		case 91:
			goto tr14
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto tr26
		}
		goto st6
tr26:
//line header.rl:15
 tokBeg = p 
	goto st16
	st16:
		if p++; p == pe {
			goto _test_eof16
		}
	st_case_16:
//line header.gen.go:487
		switch data[p] {
		case 47:
			goto tr12
		case 58:
			goto tr13
		case 91:
			goto tr14
		case 93:
			goto tr28
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st16
		}
		goto st6
tr9:
//line header.rl:21

		h.Process = data[tokBeg:p]
	
	goto st17
	st17:
		if p++; p == pe {
			goto _test_eof17
		}
	st_case_17:
//line header.gen.go:513
		if 48 <= data[p] && data[p] <= 57 {
			goto tr29
		}
		goto st0
tr29:
//line header.rl:15
 tokBeg = p 
	goto st18
	st18:
		if p++; p == pe {
			goto _test_eof18
		}
	st_case_18:
//line header.gen.go:527
		if data[p] == 93 {
			goto tr22
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st18
		}
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
	_test_eof19: cs = 19; goto _test_eof
	_test_eof10: cs = 10; goto _test_eof
	_test_eof11: cs = 11; goto _test_eof
	_test_eof12: cs = 12; goto _test_eof
	_test_eof13: cs = 13; goto _test_eof
	_test_eof20: cs = 20; goto _test_eof
	_test_eof14: cs = 14; goto _test_eof
	_test_eof21: cs = 21; goto _test_eof
	_test_eof15: cs = 15; goto _test_eof
	_test_eof16: cs = 16; goto _test_eof
	_test_eof17: cs = 17; goto _test_eof
	_test_eof18: cs = 18; goto _test_eof

	_test_eof: {}
	_out: {}
	}

//line header.rl:43


	return 0, false
}
