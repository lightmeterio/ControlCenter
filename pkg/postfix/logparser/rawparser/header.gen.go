
//line header.rl:1
// +build !codeanalysis

package rawparser


//line header.rl:6

//line header.gen.go:11
const headerPostfixPart_start int = 1
const headerPostfixPart_first_final int = 23
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
	case 2:
		goto st_case_2
	case 3:
		goto st_case_3
	case 0:
		goto st_case_0
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
	case 23:
		goto st_case_23
	case 9:
		goto st_case_9
	case 10:
		goto st_case_10
	case 11:
		goto st_case_11
	case 12:
		goto st_case_12
	case 24:
		goto st_case_24
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
	case 21:
		goto st_case_21
	case 22:
		goto st_case_22
	}
	goto st_out
	st_case_1:
		if data[p] == 32 {
			goto st0
		}
		goto tr0
tr0:
//line common.rl:17
 tokBeg = p 
	goto st2
	st2:
		if p++; p == pe {
			goto _test_eof2
		}
	st_case_2:
//line header.gen.go:105
		if data[p] == 32 {
			goto tr3
		}
		goto st2
tr3:
//line header.rl:17

		h.Host = data[tokBeg:p]
	
	goto st3
	st3:
		if p++; p == pe {
			goto _test_eof3
		}
	st_case_3:
//line header.gen.go:121
		switch data[p] {
		case 45:
			goto tr4
		case 47:
			goto st22
		}
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
st_case_0:
	st0:
		cs = 0
		goto _out
tr4:
//line common.rl:17
 tokBeg = p 
	goto st4
	st4:
		if p++; p == pe {
			goto _test_eof4
		}
	st_case_4:
//line header.gen.go:154
		switch data[p] {
		case 45:
			goto tr6
		case 47:
			goto tr7
		case 58:
			goto tr9
		case 91:
			goto tr10
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
tr6:
//line header.rl:24

		h.Process = data[tokBeg:p]
	
	goto st5
	st5:
		if p++; p == pe {
			goto _test_eof5
		}
	st_case_5:
//line header.gen.go:189
		switch data[p] {
		case 45:
			goto tr6
		case 47:
			goto tr7
		case 58:
			goto tr9
		case 91:
			goto tr10
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto tr11
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st4
			}
		default:
			goto st4
		}
		goto st0
tr7:
//line header.rl:24

		h.Process = data[tokBeg:p]
	
	goto st6
tr29:
//line header.rl:29

		h.ProcessIP = data[tokBeg:p]
	
	goto st6
	st6:
		if p++; p == pe {
			goto _test_eof6
		}
	st_case_6:
//line header.gen.go:230
		if data[p] == 93 {
			goto st0
		}
		goto tr12
tr12:
//line common.rl:17
 tokBeg = p 
	goto st7
	st7:
		if p++; p == pe {
			goto _test_eof7
		}
	st_case_7:
//line header.gen.go:244
		switch data[p] {
		case 58:
			goto tr14
		case 91:
			goto tr15
		case 93:
			goto st0
		}
		goto st7
tr14:
//line header.rl:33

		h.Daemon = data[tokBeg:p]
	
	goto st8
	st8:
		if p++; p == pe {
			goto _test_eof8
		}
	st_case_8:
//line header.gen.go:265
		switch data[p] {
		case 32:
			goto tr16
		case 58:
			goto tr14
		case 91:
			goto tr15
		case 93:
			goto st0
		}
		goto st7
tr16:
//line header.rl:43

		return p, true
	
	goto st23
	st23:
		if p++; p == pe {
			goto _test_eof23
		}
	st_case_23:
//line header.gen.go:288
		switch data[p] {
		case 58:
			goto tr14
		case 91:
			goto tr15
		case 93:
			goto st0
		}
		goto st7
tr15:
//line header.rl:33

		h.Daemon = data[tokBeg:p]
	
	goto st9
	st9:
		if p++; p == pe {
			goto _test_eof9
		}
	st_case_9:
//line header.gen.go:309
		switch data[p] {
		case 58:
			goto tr14
		case 91:
			goto tr15
		case 93:
			goto st0
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto tr17
		}
		goto st7
tr17:
//line common.rl:17
 tokBeg = p 
	goto st10
	st10:
		if p++; p == pe {
			goto _test_eof10
		}
	st_case_10:
//line header.gen.go:331
		switch data[p] {
		case 58:
			goto tr14
		case 91:
			goto tr15
		case 93:
			goto tr19
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st10
		}
		goto st7
tr19:
//line header.rl:37

		h.ProcessID = data[tokBeg:p]
	
	goto st11
	st11:
		if p++; p == pe {
			goto _test_eof11
		}
	st_case_11:
//line header.gen.go:355
		if data[p] == 58 {
			goto st12
		}
		goto st0
tr9:
//line header.rl:24

		h.Process = data[tokBeg:p]
	
	goto st12
tr30:
//line header.rl:29

		h.ProcessIP = data[tokBeg:p]
	
	goto st12
	st12:
		if p++; p == pe {
			goto _test_eof12
		}
	st_case_12:
//line header.gen.go:377
		if data[p] == 32 {
			goto tr21
		}
		goto st0
tr21:
//line header.rl:43

		return p, true
	
	goto st24
	st24:
		if p++; p == pe {
			goto _test_eof24
		}
	st_case_24:
//line header.gen.go:393
		goto st0
tr11:
//line common.rl:17
 tokBeg = p 
	goto st13
	st13:
		if p++; p == pe {
			goto _test_eof13
		}
	st_case_13:
//line header.gen.go:404
		switch data[p] {
		case 45:
			goto tr6
		case 46:
			goto st14
		case 47:
			goto tr7
		case 58:
			goto tr9
		case 91:
			goto tr10
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st13
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st4
			}
		default:
			goto st4
		}
		goto st0
	st14:
		if p++; p == pe {
			goto _test_eof14
		}
	st_case_14:
		if 48 <= data[p] && data[p] <= 57 {
			goto st15
		}
		goto st0
	st15:
		if p++; p == pe {
			goto _test_eof15
		}
	st_case_15:
		if data[p] == 46 {
			goto st16
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st15
		}
		goto st0
	st16:
		if p++; p == pe {
			goto _test_eof16
		}
	st_case_16:
		if 48 <= data[p] && data[p] <= 57 {
			goto st17
		}
		goto st0
	st17:
		if p++; p == pe {
			goto _test_eof17
		}
	st_case_17:
		if data[p] == 46 {
			goto st18
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st17
		}
		goto st0
	st18:
		if p++; p == pe {
			goto _test_eof18
		}
	st_case_18:
		if 48 <= data[p] && data[p] <= 57 {
			goto st19
		}
		goto st0
	st19:
		if p++; p == pe {
			goto _test_eof19
		}
	st_case_19:
		switch data[p] {
		case 47:
			goto tr29
		case 58:
			goto tr30
		case 91:
			goto tr31
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st19
		}
		goto st0
tr10:
//line header.rl:24

		h.Process = data[tokBeg:p]
	
	goto st20
tr31:
//line header.rl:29

		h.ProcessIP = data[tokBeg:p]
	
	goto st20
	st20:
		if p++; p == pe {
			goto _test_eof20
		}
	st_case_20:
//line header.gen.go:515
		if 48 <= data[p] && data[p] <= 57 {
			goto tr32
		}
		goto st0
tr32:
//line common.rl:17
 tokBeg = p 
	goto st21
	st21:
		if p++; p == pe {
			goto _test_eof21
		}
	st_case_21:
//line header.gen.go:529
		if data[p] == 93 {
			goto tr19
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st21
		}
		goto st0
	st22:
		if p++; p == pe {
			goto _test_eof22
		}
	st_case_22:
		if data[p] == 45 {
			goto tr4
		}
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
	st_out:
	_test_eof2: cs = 2; goto _test_eof
	_test_eof3: cs = 3; goto _test_eof
	_test_eof4: cs = 4; goto _test_eof
	_test_eof5: cs = 5; goto _test_eof
	_test_eof6: cs = 6; goto _test_eof
	_test_eof7: cs = 7; goto _test_eof
	_test_eof8: cs = 8; goto _test_eof
	_test_eof23: cs = 23; goto _test_eof
	_test_eof9: cs = 9; goto _test_eof
	_test_eof10: cs = 10; goto _test_eof
	_test_eof11: cs = 11; goto _test_eof
	_test_eof12: cs = 12; goto _test_eof
	_test_eof24: cs = 24; goto _test_eof
	_test_eof13: cs = 13; goto _test_eof
	_test_eof14: cs = 14; goto _test_eof
	_test_eof15: cs = 15; goto _test_eof
	_test_eof16: cs = 16; goto _test_eof
	_test_eof17: cs = 17; goto _test_eof
	_test_eof18: cs = 18; goto _test_eof
	_test_eof19: cs = 19; goto _test_eof
	_test_eof20: cs = 20; goto _test_eof
	_test_eof21: cs = 21; goto _test_eof
	_test_eof22: cs = 22; goto _test_eof

	_test_eof: {}
	_out: {}
	}

//line header.rl:49


	return 0, false
}
