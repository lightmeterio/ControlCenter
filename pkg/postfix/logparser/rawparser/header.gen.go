
//line header.rl:1
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !codeanalysis
// +build !codeanalysis

package rawparser


//line header.rl:11

//line header.gen.go:16
const headerPostfixPart_start int = 1
const headerPostfixPart_first_final int = 30
const headerPostfixPart_error int = 0

const headerPostfixPart_en_main int = 1


//line header.rl:12

func parseHeaderPostfixPart(h *RawHeader, data string) (int, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof


//line header.gen.go:33
	{
	cs = headerPostfixPart_start
	}

//line header.gen.go:38
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
	case 30:
		goto st_case_30
	case 9:
		goto st_case_9
	case 10:
		goto st_case_10
	case 11:
		goto st_case_11
	case 12:
		goto st_case_12
	case 31:
		goto st_case_31
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
	case 23:
		goto st_case_23
	case 24:
		goto st_case_24
	case 25:
		goto st_case_25
	case 26:
		goto st_case_26
	case 27:
		goto st_case_27
	case 28:
		goto st_case_28
	case 29:
		goto st_case_29
	}
	goto st_out
	st_case_1:
		if data[p] == 32 {
			goto st0
		}
		goto tr0
tr0:
//line common.rl:29
 tokBeg = p 
	goto st2
	st2:
		if p++; p == pe {
			goto _test_eof2
		}
	st_case_2:
//line header.gen.go:124
		if data[p] == 32 {
			goto tr3
		}
		goto st2
tr3:
//line header.rl:22

		h.Host = data[tokBeg:p]
	
	goto st3
	st3:
		if p++; p == pe {
			goto _test_eof3
		}
	st_case_3:
//line header.gen.go:140
		switch data[p] {
		case 45:
			goto tr4
		case 47:
			goto st29
		case 95:
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
st_case_0:
	st0:
		cs = 0
		goto _out
tr4:
//line common.rl:29
 tokBeg = p 
	goto st4
	st4:
		if p++; p == pe {
			goto _test_eof4
		}
	st_case_4:
//line header.gen.go:175
		switch data[p] {
		case 45:
			goto tr6
		case 47:
			goto tr7
		case 58:
			goto tr9
		case 91:
			goto tr10
		case 95:
			goto st4
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
//line header.rl:29

		h.Process = data[tokBeg:p]
	
	goto st5
	st5:
		if p++; p == pe {
			goto _test_eof5
		}
	st_case_5:
//line header.gen.go:212
		switch data[p] {
		case 45:
			goto tr6
		case 47:
			goto tr7
		case 58:
			goto tr9
		case 91:
			goto tr10
		case 95:
			goto st4
		case 117:
			goto tr12
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
//line header.rl:29

		h.Process = data[tokBeg:p]
	
	goto st6
tr30:
//line header.rl:34

		h.ProcessIP = data[tokBeg:p]
	
	goto st6
tr41:
//line header.rl:29

		h.Process = data[tokBeg:p]
	
//line header.rl:34

		h.ProcessIP = data[tokBeg:p]
	
	goto st6
	st6:
		if p++; p == pe {
			goto _test_eof6
		}
	st_case_6:
//line header.gen.go:267
		if data[p] == 93 {
			goto st0
		}
		goto tr13
tr13:
//line common.rl:29
 tokBeg = p 
	goto st7
	st7:
		if p++; p == pe {
			goto _test_eof7
		}
	st_case_7:
//line header.gen.go:281
		switch data[p] {
		case 58:
			goto tr15
		case 91:
			goto tr16
		case 93:
			goto st0
		}
		goto st7
tr15:
//line header.rl:38

		h.Daemon = data[tokBeg:p]
	
	goto st8
	st8:
		if p++; p == pe {
			goto _test_eof8
		}
	st_case_8:
//line header.gen.go:302
		switch data[p] {
		case 32:
			goto tr17
		case 58:
			goto tr15
		case 91:
			goto tr16
		case 93:
			goto st0
		}
		goto st7
tr17:
//line header.rl:48

		return p, true
	
	goto st30
	st30:
		if p++; p == pe {
			goto _test_eof30
		}
	st_case_30:
//line header.gen.go:325
		switch data[p] {
		case 58:
			goto tr15
		case 91:
			goto tr16
		case 93:
			goto st0
		}
		goto st7
tr16:
//line header.rl:38

		h.Daemon = data[tokBeg:p]
	
	goto st9
	st9:
		if p++; p == pe {
			goto _test_eof9
		}
	st_case_9:
//line header.gen.go:346
		switch data[p] {
		case 58:
			goto tr15
		case 91:
			goto tr16
		case 93:
			goto st0
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto tr18
		}
		goto st7
tr18:
//line common.rl:29
 tokBeg = p 
	goto st10
	st10:
		if p++; p == pe {
			goto _test_eof10
		}
	st_case_10:
//line header.gen.go:368
		switch data[p] {
		case 58:
			goto tr15
		case 91:
			goto tr16
		case 93:
			goto tr20
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st10
		}
		goto st7
tr20:
//line header.rl:42

		h.ProcessID = data[tokBeg:p]
	
	goto st11
	st11:
		if p++; p == pe {
			goto _test_eof11
		}
	st_case_11:
//line header.gen.go:392
		if data[p] == 58 {
			goto st12
		}
		goto st0
tr9:
//line header.rl:29

		h.Process = data[tokBeg:p]
	
	goto st12
tr31:
//line header.rl:34

		h.ProcessIP = data[tokBeg:p]
	
	goto st12
tr42:
//line header.rl:29

		h.Process = data[tokBeg:p]
	
//line header.rl:34

		h.ProcessIP = data[tokBeg:p]
	
	goto st12
	st12:
		if p++; p == pe {
			goto _test_eof12
		}
	st_case_12:
//line header.gen.go:424
		if data[p] == 32 {
			goto tr22
		}
		goto st0
tr22:
//line header.rl:48

		return p, true
	
	goto st31
	st31:
		if p++; p == pe {
			goto _test_eof31
		}
	st_case_31:
//line header.gen.go:440
		goto st0
tr11:
//line common.rl:29
 tokBeg = p 
	goto st13
	st13:
		if p++; p == pe {
			goto _test_eof13
		}
	st_case_13:
//line header.gen.go:451
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
		case 95:
			goto st4
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
			goto tr30
		case 58:
			goto tr31
		case 91:
			goto tr32
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st19
		}
		goto st0
tr10:
//line header.rl:29

		h.Process = data[tokBeg:p]
	
	goto st20
tr32:
//line header.rl:34

		h.ProcessIP = data[tokBeg:p]
	
	goto st20
tr43:
//line header.rl:29

		h.Process = data[tokBeg:p]
	
//line header.rl:34

		h.ProcessIP = data[tokBeg:p]
	
	goto st20
	st20:
		if p++; p == pe {
			goto _test_eof20
		}
	st_case_20:
//line header.gen.go:574
		if 48 <= data[p] && data[p] <= 57 {
			goto tr33
		}
		goto st0
tr33:
//line common.rl:29
 tokBeg = p 
	goto st21
	st21:
		if p++; p == pe {
			goto _test_eof21
		}
	st_case_21:
//line header.gen.go:588
		if data[p] == 93 {
			goto tr20
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st21
		}
		goto st0
tr12:
//line common.rl:29
 tokBeg = p 
	goto st22
	st22:
		if p++; p == pe {
			goto _test_eof22
		}
	st_case_22:
//line header.gen.go:605
		switch data[p] {
		case 45:
			goto tr6
		case 47:
			goto tr7
		case 58:
			goto tr9
		case 91:
			goto tr10
		case 95:
			goto st4
		case 110:
			goto st23
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
	st23:
		if p++; p == pe {
			goto _test_eof23
		}
	st_case_23:
		switch data[p] {
		case 45:
			goto tr6
		case 47:
			goto tr7
		case 58:
			goto tr9
		case 91:
			goto tr10
		case 95:
			goto st4
		case 107:
			goto st24
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
	st24:
		if p++; p == pe {
			goto _test_eof24
		}
	st_case_24:
		switch data[p] {
		case 45:
			goto tr6
		case 47:
			goto tr7
		case 58:
			goto tr9
		case 91:
			goto tr10
		case 95:
			goto st4
		case 110:
			goto st25
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
	st25:
		if p++; p == pe {
			goto _test_eof25
		}
	st_case_25:
		switch data[p] {
		case 45:
			goto tr6
		case 47:
			goto tr7
		case 58:
			goto tr9
		case 91:
			goto tr10
		case 95:
			goto st4
		case 111:
			goto st26
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
	st26:
		if p++; p == pe {
			goto _test_eof26
		}
	st_case_26:
		switch data[p] {
		case 45:
			goto tr6
		case 47:
			goto tr7
		case 58:
			goto tr9
		case 91:
			goto tr10
		case 95:
			goto st4
		case 119:
			goto st27
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
	st27:
		if p++; p == pe {
			goto _test_eof27
		}
	st_case_27:
		switch data[p] {
		case 45:
			goto tr6
		case 47:
			goto tr7
		case 58:
			goto tr9
		case 91:
			goto tr10
		case 95:
			goto st4
		case 110:
			goto st28
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
	st28:
		if p++; p == pe {
			goto _test_eof28
		}
	st_case_28:
		switch data[p] {
		case 45:
			goto tr6
		case 47:
			goto tr41
		case 58:
			goto tr42
		case 91:
			goto tr43
		case 95:
			goto st4
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
	st29:
		if p++; p == pe {
			goto _test_eof29
		}
	st_case_29:
		switch data[p] {
		case 45:
			goto tr4
		case 95:
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
	_test_eof30: cs = 30; goto _test_eof
	_test_eof9: cs = 9; goto _test_eof
	_test_eof10: cs = 10; goto _test_eof
	_test_eof11: cs = 11; goto _test_eof
	_test_eof12: cs = 12; goto _test_eof
	_test_eof31: cs = 31; goto _test_eof
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
	_test_eof23: cs = 23; goto _test_eof
	_test_eof24: cs = 24; goto _test_eof
	_test_eof25: cs = 25; goto _test_eof
	_test_eof26: cs = 26; goto _test_eof
	_test_eof27: cs = 27; goto _test_eof
	_test_eof28: cs = 28; goto _test_eof
	_test_eof29: cs = 29; goto _test_eof

	_test_eof: {}
	_out: {}
	}

//line header.rl:54


	return 0, false
}
