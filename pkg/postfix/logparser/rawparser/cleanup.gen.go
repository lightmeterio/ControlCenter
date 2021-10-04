
//line cleanup.rl:1
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !codeanalysis
// +build !codeanalysis

package rawparser


//line cleanup.rl:11

//line cleanup.gen.go:16
const cleanupMessageAccepted_start int = 1
const cleanupMessageAccepted_first_final int = 37
const cleanupMessageAccepted_error int = 0

const cleanupMessageAccepted_en_main int = 1


//line cleanup.rl:12

func parseCleanupMessageAccepted(data []byte) (CleanupMessageAccepted, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := CleanupMessageAccepted{}


//line cleanup.gen.go:35
	{
	cs = cleanupMessageAccepted_start
	}

//line cleanup.gen.go:40
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
	case 21:
		goto st_case_21
	case 22:
		goto st_case_22
	case 23:
		goto st_case_23
	case 24:
		goto st_case_24
	case 37:
		goto st_case_37
	case 38:
		goto st_case_38
	case 25:
		goto st_case_25
	case 39:
		goto st_case_39
	case 40:
		goto st_case_40
	case 41:
		goto st_case_41
	case 42:
		goto st_case_42
	case 26:
		goto st_case_26
	case 27:
		goto st_case_27
	case 28:
		goto st_case_28
	case 29:
		goto st_case_29
	case 30:
		goto st_case_30
	case 31:
		goto st_case_31
	case 32:
		goto st_case_32
	case 33:
		goto st_case_33
	case 34:
		goto st_case_34
	case 35:
		goto st_case_35
	case 36:
		goto st_case_36
	}
	goto st_out
	st_case_1:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto tr0
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto tr2
				}
			case data[p] >= 71:
				goto tr2
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
//line common.rl:29
 tokBeg = p 
	goto st2
	st2:
		if p++; p == pe {
			goto _test_eof2
		}
	st_case_2:
//line cleanup.gen.go:166
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st3
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st35
				}
			case data[p] >= 71:
				goto st35
			}
		default:
			goto st3
		}
		goto st0
	st3:
		if p++; p == pe {
			goto _test_eof3
		}
	st_case_3:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st4
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st34
				}
			case data[p] >= 71:
				goto st34
			}
		default:
			goto st4
		}
		goto st0
	st4:
		if p++; p == pe {
			goto _test_eof4
		}
	st_case_4:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st5
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st33
				}
			case data[p] >= 71:
				goto st33
			}
		default:
			goto st5
		}
		goto st0
	st5:
		if p++; p == pe {
			goto _test_eof5
		}
	st_case_5:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st6
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st32
				}
			case data[p] >= 71:
				goto st32
			}
		default:
			goto st6
		}
		goto st0
	st6:
		if p++; p == pe {
			goto _test_eof6
		}
	st_case_6:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st7
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st31
				}
			case data[p] >= 71:
				goto st31
			}
		default:
			goto st7
		}
		goto st0
	st7:
		if p++; p == pe {
			goto _test_eof7
		}
	st_case_7:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st8
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st30
				}
			case data[p] >= 71:
				goto st30
			}
		default:
			goto st8
		}
		goto st0
	st8:
		if p++; p == pe {
			goto _test_eof8
		}
	st_case_8:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st9
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st29
				}
			case data[p] >= 71:
				goto st29
			}
		default:
			goto st9
		}
		goto st0
	st9:
		if p++; p == pe {
			goto _test_eof9
		}
	st_case_9:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st10
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st28
				}
			case data[p] >= 71:
				goto st28
			}
		default:
			goto st10
		}
		goto st0
	st10:
		if p++; p == pe {
			goto _test_eof10
		}
	st_case_10:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st11
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st27
				}
			case data[p] >= 71:
				goto st27
			}
		default:
			goto st11
		}
		goto st0
	st11:
		if p++; p == pe {
			goto _test_eof11
		}
	st_case_11:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st12
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st26
				}
			case data[p] >= 71:
				goto st26
			}
		default:
			goto st12
		}
		goto st0
	st12:
		if p++; p == pe {
			goto _test_eof12
		}
	st_case_12:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st12
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st12
			}
		default:
			goto st12
		}
		goto st0
tr14:
//line cleanup.rl:24

		r.Queue = data[tokBeg:p]
	
	goto st13
	st13:
		if p++; p == pe {
			goto _test_eof13
		}
	st_case_13:
//line cleanup.gen.go:439
		if data[p] == 32 {
			goto st14
		}
		goto st0
	st14:
		if p++; p == pe {
			goto _test_eof14
		}
	st_case_14:
		if data[p] == 109 {
			goto st15
		}
		goto st0
	st15:
		if p++; p == pe {
			goto _test_eof15
		}
	st_case_15:
		if data[p] == 101 {
			goto st16
		}
		goto st0
	st16:
		if p++; p == pe {
			goto _test_eof16
		}
	st_case_16:
		if data[p] == 115 {
			goto st17
		}
		goto st0
	st17:
		if p++; p == pe {
			goto _test_eof17
		}
	st_case_17:
		if data[p] == 115 {
			goto st18
		}
		goto st0
	st18:
		if p++; p == pe {
			goto _test_eof18
		}
	st_case_18:
		if data[p] == 97 {
			goto st19
		}
		goto st0
	st19:
		if p++; p == pe {
			goto _test_eof19
		}
	st_case_19:
		if data[p] == 103 {
			goto st20
		}
		goto st0
	st20:
		if p++; p == pe {
			goto _test_eof20
		}
	st_case_20:
		if data[p] == 101 {
			goto st21
		}
		goto st0
	st21:
		if p++; p == pe {
			goto _test_eof21
		}
	st_case_21:
		if data[p] == 45 {
			goto st22
		}
		goto st0
	st22:
		if p++; p == pe {
			goto _test_eof22
		}
	st_case_22:
		if data[p] == 105 {
			goto st23
		}
		goto st0
	st23:
		if p++; p == pe {
			goto _test_eof23
		}
	st_case_23:
		if data[p] == 100 {
			goto st24
		}
		goto st0
	st24:
		if p++; p == pe {
			goto _test_eof24
		}
	st_case_24:
		if data[p] == 61 {
			goto st37
		}
		goto st0
	st37:
		if p++; p == pe {
			goto _test_eof37
		}
	st_case_37:
		switch data[p] {
		case 60:
			goto st25
		case 62:
			goto st0
		}
		goto tr38
tr38:
//line common.rl:29
 tokBeg = p 
	goto st38
	st38:
		if p++; p == pe {
			goto _test_eof38
		}
	st_case_38:
//line cleanup.gen.go:564
		switch data[p] {
		case 60:
			goto st0
		case 62:
			goto st0
		}
		goto st38
	st25:
		if p++; p == pe {
			goto _test_eof25
		}
	st_case_25:
		if data[p] == 62 {
			goto st42
		}
		goto tr36
tr36:
//line common.rl:29
 tokBeg = p 
	goto st39
	st39:
		if p++; p == pe {
			goto _test_eof39
		}
	st_case_39:
//line cleanup.gen.go:590
		if data[p] == 62 {
			goto tr42
		}
		goto st39
tr42:
//line cleanup.rl:28

    r.MessageId = data[tokBeg:p]
  
	goto st40
	st40:
		if p++; p == pe {
			goto _test_eof40
		}
	st_case_40:
//line cleanup.gen.go:606
		if data[p] == 62 {
			goto st0
		}
		goto st41
	st41:
		if p++; p == pe {
			goto _test_eof41
		}
	st_case_41:
		if data[p] == 62 {
			goto st0
		}
		goto st41
	st42:
		if p++; p == pe {
			goto _test_eof42
		}
	st_case_42:
		goto st0
	st26:
		if p++; p == pe {
			goto _test_eof26
		}
	st_case_26:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st12
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st12
			}
		default:
			goto st12
		}
		goto st0
	st27:
		if p++; p == pe {
			goto _test_eof27
		}
	st_case_27:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st26
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st26
			}
		default:
			goto st26
		}
		goto st0
	st28:
		if p++; p == pe {
			goto _test_eof28
		}
	st_case_28:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st27
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st27
			}
		default:
			goto st27
		}
		goto st0
	st29:
		if p++; p == pe {
			goto _test_eof29
		}
	st_case_29:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st28
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st28
			}
		default:
			goto st28
		}
		goto st0
	st30:
		if p++; p == pe {
			goto _test_eof30
		}
	st_case_30:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st29
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st29
			}
		default:
			goto st29
		}
		goto st0
	st31:
		if p++; p == pe {
			goto _test_eof31
		}
	st_case_31:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st30
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st30
			}
		default:
			goto st30
		}
		goto st0
	st32:
		if p++; p == pe {
			goto _test_eof32
		}
	st_case_32:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st31
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st31
			}
		default:
			goto st31
		}
		goto st0
	st33:
		if p++; p == pe {
			goto _test_eof33
		}
	st_case_33:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st32
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st32
			}
		default:
			goto st32
		}
		goto st0
	st34:
		if p++; p == pe {
			goto _test_eof34
		}
	st_case_34:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st33
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st33
			}
		default:
			goto st33
		}
		goto st0
	st35:
		if p++; p == pe {
			goto _test_eof35
		}
	st_case_35:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st34
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st34
			}
		default:
			goto st34
		}
		goto st0
tr2:
//line common.rl:29
 tokBeg = p 
	goto st36
	st36:
		if p++; p == pe {
			goto _test_eof36
		}
	st_case_36:
//line cleanup.gen.go:815
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st35
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st35
			}
		default:
			goto st35
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
	_test_eof21: cs = 21; goto _test_eof
	_test_eof22: cs = 22; goto _test_eof
	_test_eof23: cs = 23; goto _test_eof
	_test_eof24: cs = 24; goto _test_eof
	_test_eof37: cs = 37; goto _test_eof
	_test_eof38: cs = 38; goto _test_eof
	_test_eof25: cs = 25; goto _test_eof
	_test_eof39: cs = 39; goto _test_eof
	_test_eof40: cs = 40; goto _test_eof
	_test_eof41: cs = 41; goto _test_eof
	_test_eof42: cs = 42; goto _test_eof
	_test_eof26: cs = 26; goto _test_eof
	_test_eof27: cs = 27; goto _test_eof
	_test_eof28: cs = 28; goto _test_eof
	_test_eof29: cs = 29; goto _test_eof
	_test_eof30: cs = 30; goto _test_eof
	_test_eof31: cs = 31; goto _test_eof
	_test_eof32: cs = 32; goto _test_eof
	_test_eof33: cs = 33; goto _test_eof
	_test_eof34: cs = 34; goto _test_eof
	_test_eof35: cs = 35; goto _test_eof
	_test_eof36: cs = 36; goto _test_eof

	_test_eof: {}
	if p == eof {
		switch cs {
		case 40, 42:
//line cleanup.rl:42

		return r, true
	
		case 38:
//line cleanup.rl:32

    r.MessageId = data[tokBeg:p]
  
//line cleanup.rl:42

		return r, true
	
		case 37, 41:
//line cleanup.rl:38

    r.Corrupted = true
  
//line cleanup.rl:42

		return r, true
	
		case 39:
//line cleanup.rl:28

    r.MessageId = data[tokBeg:p]
  
//line cleanup.rl:38

    r.Corrupted = true
  
//line cleanup.rl:42

		return r, true
	
//line cleanup.gen.go:911
		}
	}

	_out: {}
	}

//line cleanup.rl:48


	return r, false
}


//line cleanup.rl:54

//line cleanup.gen.go:927
const cleanupMilterReject_start int = 1
const cleanupMilterReject_first_final int = 41
const cleanupMilterReject_error int = 0

const cleanupMilterReject_en_main int = 1


//line cleanup.rl:55

func parseCleanupMilterReject(data []byte) (CleanupMilterReject, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := CleanupMilterReject{}


//line cleanup.gen.go:946
	{
	cs = cleanupMilterReject_start
	}

//line cleanup.gen.go:951
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
	case 41:
		goto st_case_41
	case 30:
		goto st_case_30
	case 31:
		goto st_case_31
	case 32:
		goto st_case_32
	case 33:
		goto st_case_33
	case 34:
		goto st_case_34
	case 35:
		goto st_case_35
	case 36:
		goto st_case_36
	case 37:
		goto st_case_37
	case 38:
		goto st_case_38
	case 39:
		goto st_case_39
	case 40:
		goto st_case_40
	}
	goto st_out
	st_case_1:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto tr0
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto tr2
				}
			case data[p] >= 71:
				goto tr2
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
//line common.rl:29
 tokBeg = p 
	goto st2
	st2:
		if p++; p == pe {
			goto _test_eof2
		}
	st_case_2:
//line cleanup.gen.go:1075
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st3
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st39
				}
			case data[p] >= 71:
				goto st39
			}
		default:
			goto st3
		}
		goto st0
	st3:
		if p++; p == pe {
			goto _test_eof3
		}
	st_case_3:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st4
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st38
				}
			case data[p] >= 71:
				goto st38
			}
		default:
			goto st4
		}
		goto st0
	st4:
		if p++; p == pe {
			goto _test_eof4
		}
	st_case_4:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st5
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st37
				}
			case data[p] >= 71:
				goto st37
			}
		default:
			goto st5
		}
		goto st0
	st5:
		if p++; p == pe {
			goto _test_eof5
		}
	st_case_5:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st6
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st36
				}
			case data[p] >= 71:
				goto st36
			}
		default:
			goto st6
		}
		goto st0
	st6:
		if p++; p == pe {
			goto _test_eof6
		}
	st_case_6:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st7
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st35
				}
			case data[p] >= 71:
				goto st35
			}
		default:
			goto st7
		}
		goto st0
	st7:
		if p++; p == pe {
			goto _test_eof7
		}
	st_case_7:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st8
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st34
				}
			case data[p] >= 71:
				goto st34
			}
		default:
			goto st8
		}
		goto st0
	st8:
		if p++; p == pe {
			goto _test_eof8
		}
	st_case_8:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st9
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st33
				}
			case data[p] >= 71:
				goto st33
			}
		default:
			goto st9
		}
		goto st0
	st9:
		if p++; p == pe {
			goto _test_eof9
		}
	st_case_9:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st10
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st32
				}
			case data[p] >= 71:
				goto st32
			}
		default:
			goto st10
		}
		goto st0
	st10:
		if p++; p == pe {
			goto _test_eof10
		}
	st_case_10:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st11
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st31
				}
			case data[p] >= 71:
				goto st31
			}
		default:
			goto st11
		}
		goto st0
	st11:
		if p++; p == pe {
			goto _test_eof11
		}
	st_case_11:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st12
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st30
				}
			case data[p] >= 71:
				goto st30
			}
		default:
			goto st12
		}
		goto st0
	st12:
		if p++; p == pe {
			goto _test_eof12
		}
	st_case_12:
		if data[p] == 58 {
			goto tr14
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st12
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st12
			}
		default:
			goto st12
		}
		goto st0
tr14:
//line cleanup.rl:67

		r.Queue = data[tokBeg:p]
	
	goto st13
	st13:
		if p++; p == pe {
			goto _test_eof13
		}
	st_case_13:
//line cleanup.gen.go:1348
		if data[p] == 32 {
			goto st14
		}
		goto st0
	st14:
		if p++; p == pe {
			goto _test_eof14
		}
	st_case_14:
		if data[p] == 109 {
			goto st15
		}
		goto st0
	st15:
		if p++; p == pe {
			goto _test_eof15
		}
	st_case_15:
		if data[p] == 105 {
			goto st16
		}
		goto st0
	st16:
		if p++; p == pe {
			goto _test_eof16
		}
	st_case_16:
		if data[p] == 108 {
			goto st17
		}
		goto st0
	st17:
		if p++; p == pe {
			goto _test_eof17
		}
	st_case_17:
		if data[p] == 116 {
			goto st18
		}
		goto st0
	st18:
		if p++; p == pe {
			goto _test_eof18
		}
	st_case_18:
		if data[p] == 101 {
			goto st19
		}
		goto st0
	st19:
		if p++; p == pe {
			goto _test_eof19
		}
	st_case_19:
		if data[p] == 114 {
			goto st20
		}
		goto st0
	st20:
		if p++; p == pe {
			goto _test_eof20
		}
	st_case_20:
		if data[p] == 45 {
			goto st21
		}
		goto st0
	st21:
		if p++; p == pe {
			goto _test_eof21
		}
	st_case_21:
		if data[p] == 114 {
			goto st22
		}
		goto st0
	st22:
		if p++; p == pe {
			goto _test_eof22
		}
	st_case_22:
		if data[p] == 101 {
			goto st23
		}
		goto st0
	st23:
		if p++; p == pe {
			goto _test_eof23
		}
	st_case_23:
		if data[p] == 106 {
			goto st24
		}
		goto st0
	st24:
		if p++; p == pe {
			goto _test_eof24
		}
	st_case_24:
		if data[p] == 101 {
			goto st25
		}
		goto st0
	st25:
		if p++; p == pe {
			goto _test_eof25
		}
	st_case_25:
		if data[p] == 99 {
			goto st26
		}
		goto st0
	st26:
		if p++; p == pe {
			goto _test_eof26
		}
	st_case_26:
		if data[p] == 116 {
			goto st27
		}
		goto st0
	st27:
		if p++; p == pe {
			goto _test_eof27
		}
	st_case_27:
		if data[p] == 58 {
			goto st28
		}
		goto st0
	st28:
		if p++; p == pe {
			goto _test_eof28
		}
	st_case_28:
		if data[p] == 32 {
			goto st29
		}
		goto st0
	st29:
		if p++; p == pe {
			goto _test_eof29
		}
	st_case_29:
		goto tr40
tr40:
//line common.rl:29
 tokBeg = p 
//line cleanup.rl:73

		r.ExtraMessage = data[tokBeg:eof]
    return r, true
  
	goto st41
tr41:
//line cleanup.rl:73

		r.ExtraMessage = data[tokBeg:eof]
    return r, true
  
	goto st41
	st41:
		if p++; p == pe {
			goto _test_eof41
		}
	st_case_41:
//line cleanup.gen.go:1515
		goto tr41
	st30:
		if p++; p == pe {
			goto _test_eof30
		}
	st_case_30:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st12
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st12
			}
		default:
			goto st12
		}
		goto st0
	st31:
		if p++; p == pe {
			goto _test_eof31
		}
	st_case_31:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st30
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st30
			}
		default:
			goto st30
		}
		goto st0
	st32:
		if p++; p == pe {
			goto _test_eof32
		}
	st_case_32:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st31
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st31
			}
		default:
			goto st31
		}
		goto st0
	st33:
		if p++; p == pe {
			goto _test_eof33
		}
	st_case_33:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st32
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st32
			}
		default:
			goto st32
		}
		goto st0
	st34:
		if p++; p == pe {
			goto _test_eof34
		}
	st_case_34:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st33
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st33
			}
		default:
			goto st33
		}
		goto st0
	st35:
		if p++; p == pe {
			goto _test_eof35
		}
	st_case_35:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st34
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st34
			}
		default:
			goto st34
		}
		goto st0
	st36:
		if p++; p == pe {
			goto _test_eof36
		}
	st_case_36:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st35
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st35
			}
		default:
			goto st35
		}
		goto st0
	st37:
		if p++; p == pe {
			goto _test_eof37
		}
	st_case_37:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st36
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st36
			}
		default:
			goto st36
		}
		goto st0
	st38:
		if p++; p == pe {
			goto _test_eof38
		}
	st_case_38:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st37
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st37
			}
		default:
			goto st37
		}
		goto st0
	st39:
		if p++; p == pe {
			goto _test_eof39
		}
	st_case_39:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st38
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st38
			}
		default:
			goto st38
		}
		goto st0
tr2:
//line common.rl:29
 tokBeg = p 
	goto st40
	st40:
		if p++; p == pe {
			goto _test_eof40
		}
	st_case_40:
//line cleanup.gen.go:1706
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st39
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st39
			}
		default:
			goto st39
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
	_test_eof21: cs = 21; goto _test_eof
	_test_eof22: cs = 22; goto _test_eof
	_test_eof23: cs = 23; goto _test_eof
	_test_eof24: cs = 24; goto _test_eof
	_test_eof25: cs = 25; goto _test_eof
	_test_eof26: cs = 26; goto _test_eof
	_test_eof27: cs = 27; goto _test_eof
	_test_eof28: cs = 28; goto _test_eof
	_test_eof29: cs = 29; goto _test_eof
	_test_eof41: cs = 41; goto _test_eof
	_test_eof30: cs = 30; goto _test_eof
	_test_eof31: cs = 31; goto _test_eof
	_test_eof32: cs = 32; goto _test_eof
	_test_eof33: cs = 33; goto _test_eof
	_test_eof34: cs = 34; goto _test_eof
	_test_eof35: cs = 35; goto _test_eof
	_test_eof36: cs = 36; goto _test_eof
	_test_eof37: cs = 37; goto _test_eof
	_test_eof38: cs = 38; goto _test_eof
	_test_eof39: cs = 39; goto _test_eof
	_test_eof40: cs = 40; goto _test_eof

	_test_eof: {}
	_out: {}
	}

//line cleanup.rl:80


	return r, false
}
