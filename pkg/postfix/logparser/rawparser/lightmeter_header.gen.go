
//line lightmeter_header.rl:1
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !codeanalysis
// +build !codeanalysis

package rawparser


//line lightmeter_header.rl:11

//line lightmeter_header.gen.go:16
const lightmeter_header_start int = 1
const lightmeter_header_first_final int = 65
const lightmeter_header_error int = 0

const lightmeter_header_en_main int = 1


//line lightmeter_header.rl:12

func parseDumpedHeader(data string) (LightmeterDumpedHeader, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	var r LightmeterDumpedHeader

  var valuesTokBeg int


//line lightmeter_header.gen.go:37
	{
	cs = lightmeter_header_start
	}

//line lightmeter_header.gen.go:42
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
	case 65:
		goto st_case_65
	case 41:
		goto st_case_41
	case 42:
		goto st_case_42
	case 43:
		goto st_case_43
	case 66:
		goto st_case_66
	case 44:
		goto st_case_44
	case 45:
		goto st_case_45
	case 46:
		goto st_case_46
	case 47:
		goto st_case_47
	case 67:
		goto st_case_67
	case 48:
		goto st_case_48
	case 49:
		goto st_case_49
	case 50:
		goto st_case_50
	case 68:
		goto st_case_68
	case 51:
		goto st_case_51
	case 69:
		goto st_case_69
	case 52:
		goto st_case_52
	case 53:
		goto st_case_53
	case 54:
		goto st_case_54
	case 55:
		goto st_case_55
	case 56:
		goto st_case_56
	case 57:
		goto st_case_57
	case 58:
		goto st_case_58
	case 59:
		goto st_case_59
	case 60:
		goto st_case_60
	case 61:
		goto st_case_61
	case 62:
		goto st_case_62
	case 63:
		goto st_case_63
	case 64:
		goto st_case_64
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
//line lightmeter_header.gen.go:222
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st3
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st63
				}
			case data[p] >= 71:
				goto st63
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
					goto st62
				}
			case data[p] >= 71:
				goto st62
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
					goto st61
				}
			case data[p] >= 71:
				goto st61
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
					goto st60
				}
			case data[p] >= 71:
				goto st60
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
					goto st59
				}
			case data[p] >= 71:
				goto st59
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
					goto st58
				}
			case data[p] >= 71:
				goto st58
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
					goto st57
				}
			case data[p] >= 71:
				goto st57
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
					goto st56
				}
			case data[p] >= 71:
				goto st56
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
					goto st55
				}
			case data[p] >= 71:
				goto st55
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
					goto st54
				}
			case data[p] >= 71:
				goto st54
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
//line lightmeter_header.rl:26

    r.Queue = data[tokBeg:p]
  
	goto st13
	st13:
		if p++; p == pe {
			goto _test_eof13
		}
	st_case_13:
//line lightmeter_header.gen.go:495
		if data[p] == 32 {
			goto st14
		}
		goto st0
	st14:
		if p++; p == pe {
			goto _test_eof14
		}
	st_case_14:
		if data[p] == 104 {
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
		if data[p] == 97 {
			goto st17
		}
		goto st0
	st17:
		if p++; p == pe {
			goto _test_eof17
		}
	st_case_17:
		if data[p] == 100 {
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
		if data[p] == 32 {
			goto st21
		}
		goto st0
	st21:
		if p++; p == pe {
			goto _test_eof21
		}
	st_case_21:
		if data[p] == 110 {
			goto st22
		}
		goto st0
	st22:
		if p++; p == pe {
			goto _test_eof22
		}
	st_case_22:
		if data[p] == 97 {
			goto st23
		}
		goto st0
	st23:
		if p++; p == pe {
			goto _test_eof23
		}
	st_case_23:
		if data[p] == 109 {
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
		if data[p] == 61 {
			goto st26
		}
		goto st0
	st26:
		if p++; p == pe {
			goto _test_eof26
		}
	st_case_26:
		if data[p] == 34 {
			goto st27
		}
		goto st0
	st27:
		if p++; p == pe {
			goto _test_eof27
		}
	st_case_27:
		if data[p] == 34 {
			goto st0
		}
		goto tr38
tr38:
//line common.rl:29
 tokBeg = p 
	goto st28
	st28:
		if p++; p == pe {
			goto _test_eof28
		}
	st_case_28:
//line lightmeter_header.gen.go:635
		if data[p] == 34 {
			goto tr40
		}
		goto st28
tr40:
//line lightmeter_header.rl:30

    r.Key = data[tokBeg:p]
  
	goto st29
	st29:
		if p++; p == pe {
			goto _test_eof29
		}
	st_case_29:
//line lightmeter_header.gen.go:651
		if data[p] == 44 {
			goto st30
		}
		goto st0
	st30:
		if p++; p == pe {
			goto _test_eof30
		}
	st_case_30:
		if data[p] == 32 {
			goto st31
		}
		goto st0
	st31:
		if p++; p == pe {
			goto _test_eof31
		}
	st_case_31:
		if data[p] == 118 {
			goto st32
		}
		goto st0
	st32:
		if p++; p == pe {
			goto _test_eof32
		}
	st_case_32:
		if data[p] == 97 {
			goto st33
		}
		goto st0
	st33:
		if p++; p == pe {
			goto _test_eof33
		}
	st_case_33:
		if data[p] == 108 {
			goto st34
		}
		goto st0
	st34:
		if p++; p == pe {
			goto _test_eof34
		}
	st_case_34:
		if data[p] == 117 {
			goto st35
		}
		goto st0
	st35:
		if p++; p == pe {
			goto _test_eof35
		}
	st_case_35:
		if data[p] == 101 {
			goto st36
		}
		goto st0
	st36:
		if p++; p == pe {
			goto _test_eof36
		}
	st_case_36:
		if data[p] == 61 {
			goto st37
		}
		goto st0
	st37:
		if p++; p == pe {
			goto _test_eof37
		}
	st_case_37:
		if data[p] == 34 {
			goto st38
		}
		goto st0
	st38:
		if p++; p == pe {
			goto _test_eof38
		}
	st_case_38:
		switch data[p] {
		case 60:
			goto tr51
		case 62:
			goto st0
		}
		goto tr50
tr55:
//line common.rl:29
 tokBeg = p 
	goto st39
tr50:
//line lightmeter_header.rl:44
 valuesTokBeg = p 
//line common.rl:29
 tokBeg = p 
	goto st39
	st39:
		if p++; p == pe {
			goto _test_eof39
		}
	st_case_39:
//line lightmeter_header.gen.go:755
		switch data[p] {
		case 32:
			goto tr53
		case 34:
			goto tr54
		case 62:
			goto st0
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr53
		}
		goto st39
tr53:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
	goto st40
tr56:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
//line common.rl:29
 tokBeg = p 
	goto st40
	st40:
		if p++; p == pe {
			goto _test_eof40
		}
	st_case_40:
//line lightmeter_header.gen.go:787
		switch data[p] {
		case 32:
			goto tr56
		case 34:
			goto tr57
		case 60:
			goto tr58
		case 62:
			goto st0
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr56
		}
		goto tr55
tr54:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
//line lightmeter_header.rl:44

    r.Value = data[valuesTokBeg:p]
  
//line lightmeter_header.rl:48

		return r, true
	
	goto st65
tr57:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
//line common.rl:29
 tokBeg = p 
//line lightmeter_header.rl:44

    r.Value = data[valuesTokBeg:p]
  
//line lightmeter_header.rl:48

		return r, true
	
	goto st65
	st65:
		if p++; p == pe {
			goto _test_eof65
		}
	st_case_65:
//line lightmeter_header.gen.go:837
		switch data[p] {
		case 32:
			goto tr53
		case 34:
			goto tr54
		case 62:
			goto st0
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr53
		}
		goto st39
tr58:
//line common.rl:29
 tokBeg = p 
	goto st41
	st41:
		if p++; p == pe {
			goto _test_eof41
		}
	st_case_41:
//line lightmeter_header.gen.go:859
		switch data[p] {
		case 32:
			goto tr60
		case 34:
			goto tr61
		case 62:
			goto st0
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr60
		}
		goto tr59
tr59:
//line common.rl:29
 tokBeg = p 
	goto st42
	st42:
		if p++; p == pe {
			goto _test_eof42
		}
	st_case_42:
//line lightmeter_header.gen.go:881
		switch data[p] {
		case 32:
			goto tr63
		case 34:
			goto tr64
		case 62:
			goto tr65
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr63
		}
		goto st42
tr63:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
	goto st43
tr60:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
//line common.rl:29
 tokBeg = p 
	goto st43
tr84:
//line common.rl:29
 tokBeg = p 
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
	goto st43
	st43:
		if p++; p == pe {
			goto _test_eof43
		}
	st_case_43:
//line lightmeter_header.gen.go:921
		switch data[p] {
		case 32:
			goto tr60
		case 34:
			goto tr61
		case 60:
			goto tr66
		case 62:
			goto tr65
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr60
		}
		goto tr59
tr64:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
//line lightmeter_header.rl:44

    r.Value = data[valuesTokBeg:p]
  
//line lightmeter_header.rl:48

		return r, true
	
	goto st66
tr61:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
//line common.rl:29
 tokBeg = p 
//line lightmeter_header.rl:44

    r.Value = data[valuesTokBeg:p]
  
//line lightmeter_header.rl:48

		return r, true
	
	goto st66
tr85:
//line common.rl:29
 tokBeg = p 
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
//line lightmeter_header.rl:44

    r.Value = data[valuesTokBeg:p]
  
//line lightmeter_header.rl:48

		return r, true
	
	goto st66
	st66:
		if p++; p == pe {
			goto _test_eof66
		}
	st_case_66:
//line lightmeter_header.gen.go:987
		switch data[p] {
		case 32:
			goto tr63
		case 34:
			goto tr64
		case 62:
			goto tr65
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr63
		}
		goto st42
tr65:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
	goto st44
	st44:
		if p++; p == pe {
			goto _test_eof44
		}
	st_case_44:
//line lightmeter_header.gen.go:1011
		switch data[p] {
		case 32:
			goto st45
		case 34:
			goto tr68
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto st45
		}
		goto st0
	st45:
		if p++; p == pe {
			goto _test_eof45
		}
	st_case_45:
		switch data[p] {
		case 32:
			goto tr70
		case 60:
			goto tr71
		case 62:
			goto st0
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr70
		}
		goto tr69
tr69:
//line common.rl:29
 tokBeg = p 
	goto st46
	st46:
		if p++; p == pe {
			goto _test_eof46
		}
	st_case_46:
//line lightmeter_header.gen.go:1048
		switch data[p] {
		case 32:
			goto tr73
		case 34:
			goto tr74
		case 62:
			goto st0
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr73
		}
		goto st46
tr70:
//line common.rl:29
 tokBeg = p 
	goto st47
tr73:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
	goto st47
tr75:
//line common.rl:29
 tokBeg = p 
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
	goto st47
	st47:
		if p++; p == pe {
			goto _test_eof47
		}
	st_case_47:
//line lightmeter_header.gen.go:1084
		switch data[p] {
		case 32:
			goto tr75
		case 34:
			goto tr76
		case 60:
			goto tr71
		case 62:
			goto st0
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr75
		}
		goto tr69
tr74:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
//line lightmeter_header.rl:44

    r.Value = data[valuesTokBeg:p]
  
//line lightmeter_header.rl:48

		return r, true
	
	goto st67
tr76:
//line common.rl:29
 tokBeg = p 
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
//line lightmeter_header.rl:44

    r.Value = data[valuesTokBeg:p]
  
//line lightmeter_header.rl:48

		return r, true
	
	goto st67
	st67:
		if p++; p == pe {
			goto _test_eof67
		}
	st_case_67:
//line lightmeter_header.gen.go:1134
		switch data[p] {
		case 32:
			goto tr73
		case 34:
			goto tr74
		case 62:
			goto st0
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr73
		}
		goto st46
tr71:
//line common.rl:29
 tokBeg = p 
	goto st48
	st48:
		if p++; p == pe {
			goto _test_eof48
		}
	st_case_48:
//line lightmeter_header.gen.go:1156
		switch data[p] {
		case 32:
			goto tr78
		case 34:
			goto tr79
		case 62:
			goto st0
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr78
		}
		goto tr77
tr77:
//line common.rl:29
 tokBeg = p 
	goto st49
	st49:
		if p++; p == pe {
			goto _test_eof49
		}
	st_case_49:
//line lightmeter_header.gen.go:1178
		switch data[p] {
		case 32:
			goto tr81
		case 34:
			goto tr82
		case 62:
			goto tr65
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr81
		}
		goto st49
tr81:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
	goto st50
tr78:
//line common.rl:29
 tokBeg = p 
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
	goto st50
	st50:
		if p++; p == pe {
			goto _test_eof50
		}
	st_case_50:
//line lightmeter_header.gen.go:1210
		switch data[p] {
		case 32:
			goto tr78
		case 34:
			goto tr79
		case 60:
			goto tr83
		case 62:
			goto tr65
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr78
		}
		goto tr77
tr82:
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
//line lightmeter_header.rl:44

    r.Value = data[valuesTokBeg:p]
  
//line lightmeter_header.rl:48

		return r, true
	
	goto st68
tr79:
//line common.rl:29
 tokBeg = p 
//line lightmeter_header.rl:34

    r.Values = append(r.Values, data[tokBeg:p])
  
//line lightmeter_header.rl:44

    r.Value = data[valuesTokBeg:p]
  
//line lightmeter_header.rl:48

		return r, true
	
	goto st68
	st68:
		if p++; p == pe {
			goto _test_eof68
		}
	st_case_68:
//line lightmeter_header.gen.go:1260
		switch data[p] {
		case 32:
			goto tr81
		case 34:
			goto tr82
		case 62:
			goto tr65
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr81
		}
		goto st49
tr83:
//line common.rl:29
 tokBeg = p 
	goto st51
	st51:
		if p++; p == pe {
			goto _test_eof51
		}
	st_case_51:
//line lightmeter_header.gen.go:1282
		switch data[p] {
		case 32:
			goto tr78
		case 34:
			goto tr79
		case 62:
			goto tr65
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr78
		}
		goto tr77
tr68:
//line lightmeter_header.rl:44

    r.Value = data[valuesTokBeg:p]
  
//line lightmeter_header.rl:48

		return r, true
	
	goto st69
	st69:
		if p++; p == pe {
			goto _test_eof69
		}
	st_case_69:
//line lightmeter_header.gen.go:1310
		goto st0
tr66:
//line common.rl:29
 tokBeg = p 
	goto st52
	st52:
		if p++; p == pe {
			goto _test_eof52
		}
	st_case_52:
//line lightmeter_header.gen.go:1321
		switch data[p] {
		case 32:
			goto tr60
		case 34:
			goto tr61
		case 62:
			goto tr65
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr60
		}
		goto tr59
tr51:
//line lightmeter_header.rl:44
 valuesTokBeg = p 
//line common.rl:29
 tokBeg = p 
	goto st53
	st53:
		if p++; p == pe {
			goto _test_eof53
		}
	st_case_53:
//line lightmeter_header.gen.go:1345
		switch data[p] {
		case 32:
			goto tr84
		case 34:
			goto tr85
		case 62:
			goto st0
		}
		if 9 <= data[p] && data[p] <= 13 {
			goto tr84
		}
		goto tr59
	st54:
		if p++; p == pe {
			goto _test_eof54
		}
	st_case_54:
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
	st55:
		if p++; p == pe {
			goto _test_eof55
		}
	st_case_55:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st54
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st54
			}
		default:
			goto st54
		}
		goto st0
	st56:
		if p++; p == pe {
			goto _test_eof56
		}
	st_case_56:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st55
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st55
			}
		default:
			goto st55
		}
		goto st0
	st57:
		if p++; p == pe {
			goto _test_eof57
		}
	st_case_57:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st56
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st56
			}
		default:
			goto st56
		}
		goto st0
	st58:
		if p++; p == pe {
			goto _test_eof58
		}
	st_case_58:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st57
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st57
			}
		default:
			goto st57
		}
		goto st0
	st59:
		if p++; p == pe {
			goto _test_eof59
		}
	st_case_59:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st58
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st58
			}
		default:
			goto st58
		}
		goto st0
	st60:
		if p++; p == pe {
			goto _test_eof60
		}
	st_case_60:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st59
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st59
			}
		default:
			goto st59
		}
		goto st0
	st61:
		if p++; p == pe {
			goto _test_eof61
		}
	st_case_61:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st60
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st60
			}
		default:
			goto st60
		}
		goto st0
	st62:
		if p++; p == pe {
			goto _test_eof62
		}
	st_case_62:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st61
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st61
			}
		default:
			goto st61
		}
		goto st0
	st63:
		if p++; p == pe {
			goto _test_eof63
		}
	st_case_63:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st62
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st62
			}
		default:
			goto st62
		}
		goto st0
tr2:
//line common.rl:29
 tokBeg = p 
	goto st64
	st64:
		if p++; p == pe {
			goto _test_eof64
		}
	st_case_64:
//line lightmeter_header.gen.go:1547
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st63
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st63
			}
		default:
			goto st63
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
	_test_eof65: cs = 65; goto _test_eof
	_test_eof41: cs = 41; goto _test_eof
	_test_eof42: cs = 42; goto _test_eof
	_test_eof43: cs = 43; goto _test_eof
	_test_eof66: cs = 66; goto _test_eof
	_test_eof44: cs = 44; goto _test_eof
	_test_eof45: cs = 45; goto _test_eof
	_test_eof46: cs = 46; goto _test_eof
	_test_eof47: cs = 47; goto _test_eof
	_test_eof67: cs = 67; goto _test_eof
	_test_eof48: cs = 48; goto _test_eof
	_test_eof49: cs = 49; goto _test_eof
	_test_eof50: cs = 50; goto _test_eof
	_test_eof68: cs = 68; goto _test_eof
	_test_eof51: cs = 51; goto _test_eof
	_test_eof69: cs = 69; goto _test_eof
	_test_eof52: cs = 52; goto _test_eof
	_test_eof53: cs = 53; goto _test_eof
	_test_eof54: cs = 54; goto _test_eof
	_test_eof55: cs = 55; goto _test_eof
	_test_eof56: cs = 56; goto _test_eof
	_test_eof57: cs = 57; goto _test_eof
	_test_eof58: cs = 58; goto _test_eof
	_test_eof59: cs = 59; goto _test_eof
	_test_eof60: cs = 60; goto _test_eof
	_test_eof61: cs = 61; goto _test_eof
	_test_eof62: cs = 62; goto _test_eof
	_test_eof63: cs = 63; goto _test_eof
	_test_eof64: cs = 64; goto _test_eof

	_test_eof: {}
	_out: {}
	}

//line lightmeter_header.rl:54


	return r, false
}
