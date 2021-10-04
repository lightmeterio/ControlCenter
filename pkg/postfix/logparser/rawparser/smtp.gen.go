
//line smtp.rl:1
// SPDX-FileCopyrightText: 2021 Lightmeter <hello@lightmeter.io>
//
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !codeanalysis
// +build !codeanalysis

package rawparser

import (
	"bytes"
)


//line smtp.rl:15

//line smtp.gen.go:20
const smtpSentStatusPayload_start int = 1
const smtpSentStatusPayload_first_final int = 116
const smtpSentStatusPayload_error int = 0

const smtpSentStatusPayload_en_main int = 1


//line smtp.rl:16

func parseSmtpSentStatus(data []byte) (RawSmtpSentStatus, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := RawSmtpSentStatus{}


//line smtp.gen.go:39
	{
	cs = smtpSentStatusPayload_start
	}

//line smtp.gen.go:44
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
	case 41:
		goto st_case_41
	case 42:
		goto st_case_42
	case 43:
		goto st_case_43
	case 44:
		goto st_case_44
	case 45:
		goto st_case_45
	case 46:
		goto st_case_46
	case 47:
		goto st_case_47
	case 48:
		goto st_case_48
	case 49:
		goto st_case_49
	case 50:
		goto st_case_50
	case 51:
		goto st_case_51
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
	case 65:
		goto st_case_65
	case 66:
		goto st_case_66
	case 67:
		goto st_case_67
	case 68:
		goto st_case_68
	case 69:
		goto st_case_69
	case 70:
		goto st_case_70
	case 71:
		goto st_case_71
	case 72:
		goto st_case_72
	case 73:
		goto st_case_73
	case 74:
		goto st_case_74
	case 75:
		goto st_case_75
	case 76:
		goto st_case_76
	case 77:
		goto st_case_77
	case 78:
		goto st_case_78
	case 79:
		goto st_case_79
	case 80:
		goto st_case_80
	case 81:
		goto st_case_81
	case 82:
		goto st_case_82
	case 83:
		goto st_case_83
	case 84:
		goto st_case_84
	case 85:
		goto st_case_85
	case 86:
		goto st_case_86
	case 87:
		goto st_case_87
	case 116:
		goto st_case_116
	case 88:
		goto st_case_88
	case 89:
		goto st_case_89
	case 90:
		goto st_case_90
	case 91:
		goto st_case_91
	case 92:
		goto st_case_92
	case 93:
		goto st_case_93
	case 94:
		goto st_case_94
	case 95:
		goto st_case_95
	case 96:
		goto st_case_96
	case 97:
		goto st_case_97
	case 98:
		goto st_case_98
	case 99:
		goto st_case_99
	case 100:
		goto st_case_100
	case 101:
		goto st_case_101
	case 102:
		goto st_case_102
	case 103:
		goto st_case_103
	case 104:
		goto st_case_104
	case 105:
		goto st_case_105
	case 106:
		goto st_case_106
	case 107:
		goto st_case_107
	case 108:
		goto st_case_108
	case 109:
		goto st_case_109
	case 110:
		goto st_case_110
	case 111:
		goto st_case_111
	case 112:
		goto st_case_112
	case 113:
		goto st_case_113
	case 114:
		goto st_case_114
	case 115:
		goto st_case_115
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
//line smtp.gen.go:318
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st3
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st114
				}
			case data[p] >= 71:
				goto st114
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
					goto st113
				}
			case data[p] >= 71:
				goto st113
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
					goto st112
				}
			case data[p] >= 71:
				goto st112
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
					goto st111
				}
			case data[p] >= 71:
				goto st111
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
					goto st110
				}
			case data[p] >= 71:
				goto st110
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
					goto st109
				}
			case data[p] >= 71:
				goto st109
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
					goto st108
				}
			case data[p] >= 71:
				goto st108
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
					goto st107
				}
			case data[p] >= 71:
				goto st107
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
					goto st106
				}
			case data[p] >= 71:
				goto st106
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
					goto st105
				}
			case data[p] >= 71:
				goto st105
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
//line smtp.rl:28

		r.Queue = data[tokBeg:p]
	
	goto st13
	st13:
		if p++; p == pe {
			goto _test_eof13
		}
	st_case_13:
//line smtp.gen.go:591
		if data[p] == 32 {
			goto st14
		}
		goto st0
	st14:
		if p++; p == pe {
			goto _test_eof14
		}
	st_case_14:
		if data[p] == 116 {
			goto st15
		}
		goto st0
	st15:
		if p++; p == pe {
			goto _test_eof15
		}
	st_case_15:
		if data[p] == 111 {
			goto st16
		}
		goto st0
	st16:
		if p++; p == pe {
			goto _test_eof16
		}
	st_case_16:
		if data[p] == 61 {
			goto st17
		}
		goto st0
	st17:
		if p++; p == pe {
			goto _test_eof17
		}
	st_case_17:
		if data[p] == 60 {
			goto st18
		}
		goto st0
	st18:
		if p++; p == pe {
			goto _test_eof18
		}
	st_case_18:
		switch data[p] {
		case 62:
			goto st0
		case 64:
			goto st0
		}
		goto tr29
tr29:
//line common.rl:29
 tokBeg = p 
	goto st19
	st19:
		if p++; p == pe {
			goto _test_eof19
		}
	st_case_19:
//line smtp.gen.go:653
		switch data[p] {
		case 62:
			goto st0
		case 64:
			goto tr31
		}
		goto st19
tr31:
//line smtp.rl:32

		r.RecipientLocalPart = normalizeMailLocalPart(data[tokBeg:p])
	
	goto st20
	st20:
		if p++; p == pe {
			goto _test_eof20
		}
	st_case_20:
//line smtp.gen.go:672
		if data[p] == 62 {
			goto st0
		}
		goto tr32
tr32:
//line common.rl:29
 tokBeg = p 
	goto st21
	st21:
		if p++; p == pe {
			goto _test_eof21
		}
	st_case_21:
//line smtp.gen.go:686
		if data[p] == 62 {
			goto tr34
		}
		goto st21
tr34:
//line smtp.rl:36

		r.RecipientDomainPart = data[tokBeg:p]
	
	goto st22
	st22:
		if p++; p == pe {
			goto _test_eof22
		}
	st_case_22:
//line smtp.gen.go:702
		if data[p] == 44 {
			goto st23
		}
		goto st0
	st23:
		if p++; p == pe {
			goto _test_eof23
		}
	st_case_23:
		if data[p] == 32 {
			goto st24
		}
		goto st0
	st24:
		if p++; p == pe {
			goto _test_eof24
		}
	st_case_24:
		switch data[p] {
		case 111:
			goto st25
		case 114:
			goto st38
		}
		goto st0
	st25:
		if p++; p == pe {
			goto _test_eof25
		}
	st_case_25:
		if data[p] == 114 {
			goto st26
		}
		goto st0
	st26:
		if p++; p == pe {
			goto _test_eof26
		}
	st_case_26:
		if data[p] == 105 {
			goto st27
		}
		goto st0
	st27:
		if p++; p == pe {
			goto _test_eof27
		}
	st_case_27:
		if data[p] == 103 {
			goto st28
		}
		goto st0
	st28:
		if p++; p == pe {
			goto _test_eof28
		}
	st_case_28:
		if data[p] == 95 {
			goto st29
		}
		goto st0
	st29:
		if p++; p == pe {
			goto _test_eof29
		}
	st_case_29:
		if data[p] == 116 {
			goto st30
		}
		goto st0
	st30:
		if p++; p == pe {
			goto _test_eof30
		}
	st_case_30:
		if data[p] == 111 {
			goto st31
		}
		goto st0
	st31:
		if p++; p == pe {
			goto _test_eof31
		}
	st_case_31:
		if data[p] == 61 {
			goto st32
		}
		goto st0
	st32:
		if p++; p == pe {
			goto _test_eof32
		}
	st_case_32:
		if data[p] == 60 {
			goto st33
		}
		goto st0
	st33:
		if p++; p == pe {
			goto _test_eof33
		}
	st_case_33:
		switch data[p] {
		case 62:
			goto st0
		case 64:
			goto st0
		}
		goto tr47
tr47:
//line common.rl:29
 tokBeg = p 
	goto st34
	st34:
		if p++; p == pe {
			goto _test_eof34
		}
	st_case_34:
//line smtp.gen.go:821
		switch data[p] {
		case 62:
			goto tr49
		case 64:
			goto tr50
		}
		goto st34
tr49:
//line smtp.rl:40

		r.OrigRecipientLocalPart = normalizeMailLocalPart(data[tokBeg:p])
	
	goto st35
tr130:
//line smtp.rl:44

		r.OrigRecipientDomainPart = data[tokBeg:p]
	
	goto st35
	st35:
		if p++; p == pe {
			goto _test_eof35
		}
	st_case_35:
//line smtp.gen.go:846
		if data[p] == 44 {
			goto st36
		}
		goto st0
	st36:
		if p++; p == pe {
			goto _test_eof36
		}
	st_case_36:
		if data[p] == 32 {
			goto st37
		}
		goto st0
	st37:
		if p++; p == pe {
			goto _test_eof37
		}
	st_case_37:
		if data[p] == 114 {
			goto st38
		}
		goto st0
	st38:
		if p++; p == pe {
			goto _test_eof38
		}
	st_case_38:
		if data[p] == 101 {
			goto st39
		}
		goto st0
	st39:
		if p++; p == pe {
			goto _test_eof39
		}
	st_case_39:
		if data[p] == 108 {
			goto st40
		}
		goto st0
	st40:
		if p++; p == pe {
			goto _test_eof40
		}
	st_case_40:
		if data[p] == 97 {
			goto st41
		}
		goto st0
	st41:
		if p++; p == pe {
			goto _test_eof41
		}
	st_case_41:
		if data[p] == 121 {
			goto st42
		}
		goto st0
	st42:
		if p++; p == pe {
			goto _test_eof42
		}
	st_case_42:
		if data[p] == 61 {
			goto st43
		}
		goto st0
	st43:
		if p++; p == pe {
			goto _test_eof43
		}
	st_case_43:
		switch data[p] {
		case 44:
			goto st0
		case 91:
			goto st0
		}
		goto tr58
tr58:
//line common.rl:29
 tokBeg = p 
	goto st44
	st44:
		if p++; p == pe {
			goto _test_eof44
		}
	st_case_44:
//line smtp.gen.go:935
		switch data[p] {
		case 44:
			goto tr60
		case 58:
			goto tr61
		case 91:
			goto tr62
		}
		goto st44
tr60:
//line smtp.rl:48

		r.RelayName = data[tokBeg:p]
	
	goto st45
tr118:
//line smtp.rl:48

		r.RelayName = data[tokBeg:p]
	
//line smtp.rl:56

		r.RelayPort = data[tokBeg:p]
	
	goto st45
tr126:
//line smtp.rl:56

		r.RelayPort = data[tokBeg:p]
	
	goto st45
	st45:
		if p++; p == pe {
			goto _test_eof45
		}
	st_case_45:
//line smtp.gen.go:972
		if data[p] == 32 {
			goto st46
		}
		goto st0
	st46:
		if p++; p == pe {
			goto _test_eof46
		}
	st_case_46:
		if data[p] == 100 {
			goto st47
		}
		goto st0
	st47:
		if p++; p == pe {
			goto _test_eof47
		}
	st_case_47:
		if data[p] == 101 {
			goto st48
		}
		goto st0
	st48:
		if p++; p == pe {
			goto _test_eof48
		}
	st_case_48:
		if data[p] == 108 {
			goto st49
		}
		goto st0
	st49:
		if p++; p == pe {
			goto _test_eof49
		}
	st_case_49:
		if data[p] == 97 {
			goto st50
		}
		goto st0
	st50:
		if p++; p == pe {
			goto _test_eof50
		}
	st_case_50:
		if data[p] == 121 {
			goto st51
		}
		goto st0
	st51:
		if p++; p == pe {
			goto _test_eof51
		}
	st_case_51:
		if data[p] == 61 {
			goto st52
		}
		goto st0
	st52:
		if p++; p == pe {
			goto _test_eof52
		}
	st_case_52:
		if data[p] == 44 {
			goto st0
		}
		goto tr70
tr70:
//line common.rl:29
 tokBeg = p 
	goto st53
	st53:
		if p++; p == pe {
			goto _test_eof53
		}
	st_case_53:
//line smtp.gen.go:1049
		if data[p] == 44 {
			goto tr72
		}
		goto st53
tr72:
//line smtp.rl:60

		r.Delay = data[tokBeg:p]
	
	goto st54
	st54:
		if p++; p == pe {
			goto _test_eof54
		}
	st_case_54:
//line smtp.gen.go:1065
		if data[p] == 32 {
			goto st55
		}
		goto st0
	st55:
		if p++; p == pe {
			goto _test_eof55
		}
	st_case_55:
		if data[p] == 100 {
			goto st56
		}
		goto st0
	st56:
		if p++; p == pe {
			goto _test_eof56
		}
	st_case_56:
		if data[p] == 101 {
			goto st57
		}
		goto st0
	st57:
		if p++; p == pe {
			goto _test_eof57
		}
	st_case_57:
		if data[p] == 108 {
			goto st58
		}
		goto st0
	st58:
		if p++; p == pe {
			goto _test_eof58
		}
	st_case_58:
		if data[p] == 97 {
			goto st59
		}
		goto st0
	st59:
		if p++; p == pe {
			goto _test_eof59
		}
	st_case_59:
		if data[p] == 121 {
			goto st60
		}
		goto st0
	st60:
		if p++; p == pe {
			goto _test_eof60
		}
	st_case_60:
		if data[p] == 115 {
			goto st61
		}
		goto st0
	st61:
		if p++; p == pe {
			goto _test_eof61
		}
	st_case_61:
		if data[p] == 61 {
			goto st62
		}
		goto st0
	st62:
		if p++; p == pe {
			goto _test_eof62
		}
	st_case_62:
		if data[p] == 44 {
			goto st0
		}
		goto tr81
tr81:
//line common.rl:29
 tokBeg = p 
	goto st63
	st63:
		if p++; p == pe {
			goto _test_eof63
		}
	st_case_63:
//line smtp.gen.go:1151
		if data[p] == 44 {
			goto tr83
		}
		goto st63
tr83:
//line smtp.rl:64

		{
			delays := data[tokBeg:p]
			split := bytes.Split(delays, []byte("/"))

			if len(split) != 4 {
				// delays has format 0.0/0.0/0.0/0.0
				return r, false
			}

			r.Delays[0] = delays
			r.Delays[1] = split[0]
			r.Delays[2] = split[1]
			r.Delays[3] = split[2]
			r.Delays[4] = split[3]
		}
	
	goto st64
	st64:
		if p++; p == pe {
			goto _test_eof64
		}
	st_case_64:
//line smtp.gen.go:1181
		if data[p] == 32 {
			goto st65
		}
		goto st0
	st65:
		if p++; p == pe {
			goto _test_eof65
		}
	st_case_65:
		if data[p] == 100 {
			goto st66
		}
		goto st0
	st66:
		if p++; p == pe {
			goto _test_eof66
		}
	st_case_66:
		if data[p] == 115 {
			goto st67
		}
		goto st0
	st67:
		if p++; p == pe {
			goto _test_eof67
		}
	st_case_67:
		if data[p] == 110 {
			goto st68
		}
		goto st0
	st68:
		if p++; p == pe {
			goto _test_eof68
		}
	st_case_68:
		if data[p] == 61 {
			goto st69
		}
		goto st0
	st69:
		if p++; p == pe {
			goto _test_eof69
		}
	st_case_69:
		if data[p] == 44 {
			goto st0
		}
		goto tr89
tr89:
//line common.rl:29
 tokBeg = p 
	goto st70
	st70:
		if p++; p == pe {
			goto _test_eof70
		}
	st_case_70:
//line smtp.gen.go:1240
		if data[p] == 44 {
			goto tr91
		}
		goto st70
tr91:
//line smtp.rl:82

		r.Dsn = data[tokBeg:p]
	
	goto st71
	st71:
		if p++; p == pe {
			goto _test_eof71
		}
	st_case_71:
//line smtp.gen.go:1256
		if data[p] == 32 {
			goto st72
		}
		goto st0
	st72:
		if p++; p == pe {
			goto _test_eof72
		}
	st_case_72:
		if data[p] == 115 {
			goto st73
		}
		goto st0
	st73:
		if p++; p == pe {
			goto _test_eof73
		}
	st_case_73:
		if data[p] == 116 {
			goto st74
		}
		goto st0
	st74:
		if p++; p == pe {
			goto _test_eof74
		}
	st_case_74:
		if data[p] == 97 {
			goto st75
		}
		goto st0
	st75:
		if p++; p == pe {
			goto _test_eof75
		}
	st_case_75:
		if data[p] == 116 {
			goto st76
		}
		goto st0
	st76:
		if p++; p == pe {
			goto _test_eof76
		}
	st_case_76:
		if data[p] == 117 {
			goto st77
		}
		goto st0
	st77:
		if p++; p == pe {
			goto _test_eof77
		}
	st_case_77:
		if data[p] == 115 {
			goto st78
		}
		goto st0
	st78:
		if p++; p == pe {
			goto _test_eof78
		}
	st_case_78:
		if data[p] == 61 {
			goto st79
		}
		goto st0
	st79:
		if p++; p == pe {
			goto _test_eof79
		}
	st_case_79:
		switch data[p] {
		case 98:
			goto tr100
		case 100:
			goto tr101
		case 115:
			goto tr102
		}
		goto st0
tr100:
//line common.rl:29
 tokBeg = p 
	goto st80
	st80:
		if p++; p == pe {
			goto _test_eof80
		}
	st_case_80:
//line smtp.gen.go:1347
		if data[p] == 111 {
			goto st81
		}
		goto st0
	st81:
		if p++; p == pe {
			goto _test_eof81
		}
	st_case_81:
		if data[p] == 117 {
			goto st82
		}
		goto st0
	st82:
		if p++; p == pe {
			goto _test_eof82
		}
	st_case_82:
		if data[p] == 110 {
			goto st83
		}
		goto st0
	st83:
		if p++; p == pe {
			goto _test_eof83
		}
	st_case_83:
		if data[p] == 99 {
			goto st84
		}
		goto st0
	st84:
		if p++; p == pe {
			goto _test_eof84
		}
	st_case_84:
		if data[p] == 101 {
			goto st85
		}
		goto st0
	st85:
		if p++; p == pe {
			goto _test_eof85
		}
	st_case_85:
		if data[p] == 100 {
			goto st86
		}
		goto st0
	st86:
		if p++; p == pe {
			goto _test_eof86
		}
	st_case_86:
		if data[p] == 32 {
			goto tr109
		}
		goto st0
tr109:
//line smtp.rl:86

		r.Status = data[tokBeg:p]
	
	goto st87
	st87:
		if p++; p == pe {
			goto _test_eof87
		}
	st_case_87:
//line smtp.gen.go:1417
		goto tr110
tr110:
//line common.rl:29
 tokBeg = p 
//line smtp.rl:95

		r.ExtraMessage = data[tokBeg:eof]
		return r, true
	
	goto st116
tr131:
//line smtp.rl:95

		r.ExtraMessage = data[tokBeg:eof]
		return r, true
	
	goto st116
	st116:
		if p++; p == pe {
			goto _test_eof116
		}
	st_case_116:
//line smtp.gen.go:1440
		goto tr131
tr101:
//line common.rl:29
 tokBeg = p 
	goto st88
	st88:
		if p++; p == pe {
			goto _test_eof88
		}
	st_case_88:
//line smtp.gen.go:1451
		if data[p] == 101 {
			goto st89
		}
		goto st0
	st89:
		if p++; p == pe {
			goto _test_eof89
		}
	st_case_89:
		if data[p] == 102 {
			goto st90
		}
		goto st0
	st90:
		if p++; p == pe {
			goto _test_eof90
		}
	st_case_90:
		if data[p] == 101 {
			goto st91
		}
		goto st0
	st91:
		if p++; p == pe {
			goto _test_eof91
		}
	st_case_91:
		if data[p] == 114 {
			goto st92
		}
		goto st0
	st92:
		if p++; p == pe {
			goto _test_eof92
		}
	st_case_92:
		if data[p] == 114 {
			goto st84
		}
		goto st0
tr102:
//line common.rl:29
 tokBeg = p 
	goto st93
	st93:
		if p++; p == pe {
			goto _test_eof93
		}
	st_case_93:
//line smtp.gen.go:1501
		if data[p] == 101 {
			goto st94
		}
		goto st0
	st94:
		if p++; p == pe {
			goto _test_eof94
		}
	st_case_94:
		if data[p] == 110 {
			goto st95
		}
		goto st0
	st95:
		if p++; p == pe {
			goto _test_eof95
		}
	st_case_95:
		if data[p] == 116 {
			goto st86
		}
		goto st0
tr61:
//line smtp.rl:48

		r.RelayName = data[tokBeg:p]
	
	goto st96
	st96:
		if p++; p == pe {
			goto _test_eof96
		}
	st_case_96:
//line smtp.gen.go:1535
		switch data[p] {
		case 44:
			goto tr60
		case 58:
			goto tr61
		case 91:
			goto tr62
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto tr117
		}
		goto st44
tr117:
//line common.rl:29
 tokBeg = p 
	goto st97
	st97:
		if p++; p == pe {
			goto _test_eof97
		}
	st_case_97:
//line smtp.gen.go:1557
		switch data[p] {
		case 44:
			goto tr118
		case 58:
			goto tr61
		case 91:
			goto tr62
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st97
		}
		goto st44
tr62:
//line smtp.rl:48

		r.RelayName = data[tokBeg:p]
	
	goto st98
	st98:
		if p++; p == pe {
			goto _test_eof98
		}
	st_case_98:
//line smtp.gen.go:1581
		if data[p] == 93 {
			goto st0
		}
		goto tr120
tr120:
//line common.rl:29
 tokBeg = p 
	goto st99
	st99:
		if p++; p == pe {
			goto _test_eof99
		}
	st_case_99:
//line smtp.gen.go:1595
		if data[p] == 93 {
			goto tr122
		}
		goto st99
tr122:
//line smtp.rl:52

		r.RelayIpOrPath = data[tokBeg:p]
	
	goto st100
	st100:
		if p++; p == pe {
			goto _test_eof100
		}
	st_case_100:
//line smtp.gen.go:1611
		switch data[p] {
		case 44:
			goto st45
		case 58:
			goto st101
		}
		goto st0
	st101:
		if p++; p == pe {
			goto _test_eof101
		}
	st_case_101:
		if 48 <= data[p] && data[p] <= 57 {
			goto tr125
		}
		goto st0
tr125:
//line common.rl:29
 tokBeg = p 
	goto st102
	st102:
		if p++; p == pe {
			goto _test_eof102
		}
	st_case_102:
//line smtp.gen.go:1637
		if data[p] == 44 {
			goto tr126
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st102
		}
		goto st0
tr50:
//line smtp.rl:40

		r.OrigRecipientLocalPart = normalizeMailLocalPart(data[tokBeg:p])
	
	goto st103
	st103:
		if p++; p == pe {
			goto _test_eof103
		}
	st_case_103:
//line smtp.gen.go:1656
		if data[p] == 62 {
			goto st0
		}
		goto tr128
tr128:
//line common.rl:29
 tokBeg = p 
	goto st104
	st104:
		if p++; p == pe {
			goto _test_eof104
		}
	st_case_104:
//line smtp.gen.go:1670
		if data[p] == 62 {
			goto tr130
		}
		goto st104
	st105:
		if p++; p == pe {
			goto _test_eof105
		}
	st_case_105:
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
	st106:
		if p++; p == pe {
			goto _test_eof106
		}
	st_case_106:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st105
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st105
			}
		default:
			goto st105
		}
		goto st0
	st107:
		if p++; p == pe {
			goto _test_eof107
		}
	st_case_107:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st106
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st106
			}
		default:
			goto st106
		}
		goto st0
	st108:
		if p++; p == pe {
			goto _test_eof108
		}
	st_case_108:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st107
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st107
			}
		default:
			goto st107
		}
		goto st0
	st109:
		if p++; p == pe {
			goto _test_eof109
		}
	st_case_109:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st108
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st108
			}
		default:
			goto st108
		}
		goto st0
	st110:
		if p++; p == pe {
			goto _test_eof110
		}
	st_case_110:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st109
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st109
			}
		default:
			goto st109
		}
		goto st0
	st111:
		if p++; p == pe {
			goto _test_eof111
		}
	st_case_111:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st110
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st110
			}
		default:
			goto st110
		}
		goto st0
	st112:
		if p++; p == pe {
			goto _test_eof112
		}
	st_case_112:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st111
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st111
			}
		default:
			goto st111
		}
		goto st0
	st113:
		if p++; p == pe {
			goto _test_eof113
		}
	st_case_113:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st112
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st112
			}
		default:
			goto st112
		}
		goto st0
	st114:
		if p++; p == pe {
			goto _test_eof114
		}
	st_case_114:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st113
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st113
			}
		default:
			goto st113
		}
		goto st0
tr2:
//line common.rl:29
 tokBeg = p 
	goto st115
	st115:
		if p++; p == pe {
			goto _test_eof115
		}
	st_case_115:
//line smtp.gen.go:1864
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st114
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st114
			}
		default:
			goto st114
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
	_test_eof41: cs = 41; goto _test_eof
	_test_eof42: cs = 42; goto _test_eof
	_test_eof43: cs = 43; goto _test_eof
	_test_eof44: cs = 44; goto _test_eof
	_test_eof45: cs = 45; goto _test_eof
	_test_eof46: cs = 46; goto _test_eof
	_test_eof47: cs = 47; goto _test_eof
	_test_eof48: cs = 48; goto _test_eof
	_test_eof49: cs = 49; goto _test_eof
	_test_eof50: cs = 50; goto _test_eof
	_test_eof51: cs = 51; goto _test_eof
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
	_test_eof65: cs = 65; goto _test_eof
	_test_eof66: cs = 66; goto _test_eof
	_test_eof67: cs = 67; goto _test_eof
	_test_eof68: cs = 68; goto _test_eof
	_test_eof69: cs = 69; goto _test_eof
	_test_eof70: cs = 70; goto _test_eof
	_test_eof71: cs = 71; goto _test_eof
	_test_eof72: cs = 72; goto _test_eof
	_test_eof73: cs = 73; goto _test_eof
	_test_eof74: cs = 74; goto _test_eof
	_test_eof75: cs = 75; goto _test_eof
	_test_eof76: cs = 76; goto _test_eof
	_test_eof77: cs = 77; goto _test_eof
	_test_eof78: cs = 78; goto _test_eof
	_test_eof79: cs = 79; goto _test_eof
	_test_eof80: cs = 80; goto _test_eof
	_test_eof81: cs = 81; goto _test_eof
	_test_eof82: cs = 82; goto _test_eof
	_test_eof83: cs = 83; goto _test_eof
	_test_eof84: cs = 84; goto _test_eof
	_test_eof85: cs = 85; goto _test_eof
	_test_eof86: cs = 86; goto _test_eof
	_test_eof87: cs = 87; goto _test_eof
	_test_eof116: cs = 116; goto _test_eof
	_test_eof88: cs = 88; goto _test_eof
	_test_eof89: cs = 89; goto _test_eof
	_test_eof90: cs = 90; goto _test_eof
	_test_eof91: cs = 91; goto _test_eof
	_test_eof92: cs = 92; goto _test_eof
	_test_eof93: cs = 93; goto _test_eof
	_test_eof94: cs = 94; goto _test_eof
	_test_eof95: cs = 95; goto _test_eof
	_test_eof96: cs = 96; goto _test_eof
	_test_eof97: cs = 97; goto _test_eof
	_test_eof98: cs = 98; goto _test_eof
	_test_eof99: cs = 99; goto _test_eof
	_test_eof100: cs = 100; goto _test_eof
	_test_eof101: cs = 101; goto _test_eof
	_test_eof102: cs = 102; goto _test_eof
	_test_eof103: cs = 103; goto _test_eof
	_test_eof104: cs = 104; goto _test_eof
	_test_eof105: cs = 105; goto _test_eof
	_test_eof106: cs = 106; goto _test_eof
	_test_eof107: cs = 107; goto _test_eof
	_test_eof108: cs = 108; goto _test_eof
	_test_eof109: cs = 109; goto _test_eof
	_test_eof110: cs = 110; goto _test_eof
	_test_eof111: cs = 111; goto _test_eof
	_test_eof112: cs = 112; goto _test_eof
	_test_eof113: cs = 113; goto _test_eof
	_test_eof114: cs = 114; goto _test_eof
	_test_eof115: cs = 115; goto _test_eof

	_test_eof: {}
	_out: {}
	}

//line smtp.rl:102


	return r, false
}


//line smtp.rl:108

//line smtp.gen.go:2008
const smtpSentStatusExtraMessageSentQueuedPayload_start int = 1
const smtpSentStatusExtraMessageSentQueuedPayload_first_final int = 92
const smtpSentStatusExtraMessageSentQueuedPayload_error int = 0

const smtpSentStatusExtraMessageSentQueuedPayload_en_main int = 1


//line smtp.rl:109

func parseSmtpSentStatusExtraMessageSentQueued(data []byte) (SmtpSentStatusExtraMessageSentQueued, bool) {
	cs, p, pe, eof := 0, 0, len(data), len(data)
	tokBeg := 0

	_ = eof

	r := SmtpSentStatusExtraMessageSentQueued{}


//line smtp.gen.go:2027
	{
	cs = smtpSentStatusExtraMessageSentQueuedPayload_start
	}

//line smtp.gen.go:2032
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
	case 41:
		goto st_case_41
	case 42:
		goto st_case_42
	case 43:
		goto st_case_43
	case 44:
		goto st_case_44
	case 45:
		goto st_case_45
	case 46:
		goto st_case_46
	case 47:
		goto st_case_47
	case 48:
		goto st_case_48
	case 49:
		goto st_case_49
	case 50:
		goto st_case_50
	case 51:
		goto st_case_51
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
	case 65:
		goto st_case_65
	case 66:
		goto st_case_66
	case 67:
		goto st_case_67
	case 68:
		goto st_case_68
	case 92:
		goto st_case_92
	case 69:
		goto st_case_69
	case 70:
		goto st_case_70
	case 71:
		goto st_case_71
	case 72:
		goto st_case_72
	case 73:
		goto st_case_73
	case 74:
		goto st_case_74
	case 75:
		goto st_case_75
	case 76:
		goto st_case_76
	case 77:
		goto st_case_77
	case 78:
		goto st_case_78
	case 79:
		goto st_case_79
	case 80:
		goto st_case_80
	case 81:
		goto st_case_81
	case 82:
		goto st_case_82
	case 83:
		goto st_case_83
	case 84:
		goto st_case_84
	case 85:
		goto st_case_85
	case 86:
		goto st_case_86
	case 87:
		goto st_case_87
	case 88:
		goto st_case_88
	case 89:
		goto st_case_89
	case 90:
		goto st_case_90
	case 91:
		goto st_case_91
	}
	goto st_out
	st_case_1:
		if data[p] == 40 {
			goto st2
		}
		goto st0
st_case_0:
	st0:
		cs = 0
		goto _out
	st2:
		if p++; p == pe {
			goto _test_eof2
		}
	st_case_2:
		if 48 <= data[p] && data[p] <= 57 {
			goto tr2
		}
		goto st0
tr2:
//line common.rl:29
 tokBeg = p 
	goto st3
	st3:
		if p++; p == pe {
			goto _test_eof3
		}
	st_case_3:
//line smtp.gen.go:2253
		if data[p] == 32 {
			goto tr3
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st3
		}
		goto st0
tr3:
//line smtp.rl:121

		r.SmtpCode = data[tokBeg:p]
	
	goto st4
	st4:
		if p++; p == pe {
			goto _test_eof4
		}
	st_case_4:
//line smtp.gen.go:2272
		if 48 <= data[p] && data[p] <= 57 {
			goto tr5
		}
		goto st0
tr5:
//line common.rl:29
 tokBeg = p 
	goto st5
	st5:
		if p++; p == pe {
			goto _test_eof5
		}
	st_case_5:
//line smtp.gen.go:2286
		if data[p] == 46 {
			goto st6
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st5
		}
		goto st0
	st6:
		if p++; p == pe {
			goto _test_eof6
		}
	st_case_6:
		if 48 <= data[p] && data[p] <= 57 {
			goto st7
		}
		goto st0
	st7:
		if p++; p == pe {
			goto _test_eof7
		}
	st_case_7:
		if data[p] == 46 {
			goto st8
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st7
		}
		goto st0
	st8:
		if p++; p == pe {
			goto _test_eof8
		}
	st_case_8:
		if 48 <= data[p] && data[p] <= 57 {
			goto st9
		}
		goto st0
	st9:
		if p++; p == pe {
			goto _test_eof9
		}
	st_case_9:
		if data[p] == 32 {
			goto tr11
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st9
		}
		goto st0
tr11:
//line smtp.rl:125

		r.Dsn = data[tokBeg:p]
	
	goto st10
	st10:
		if p++; p == pe {
			goto _test_eof10
		}
	st_case_10:
//line smtp.gen.go:2347
		if data[p] == 102 {
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
		if data[p] == 111 {
			goto st13
		}
		goto st0
	st13:
		if p++; p == pe {
			goto _test_eof13
		}
	st_case_13:
		if data[p] == 109 {
			goto st14
		}
		goto st0
	st14:
		if p++; p == pe {
			goto _test_eof14
		}
	st_case_14:
		if data[p] == 32 {
			goto st15
		}
		goto st0
	st15:
		if p++; p == pe {
			goto _test_eof15
		}
	st_case_15:
		if data[p] == 77 {
			goto st16
		}
		goto st0
	st16:
		if p++; p == pe {
			goto _test_eof16
		}
	st_case_16:
		if data[p] == 84 {
			goto st17
		}
		goto st0
	st17:
		if p++; p == pe {
			goto _test_eof17
		}
	st_case_17:
		if data[p] == 65 {
			goto st18
		}
		goto st0
	st18:
		if p++; p == pe {
			goto _test_eof18
		}
	st_case_18:
		if data[p] == 40 {
			goto st19
		}
		goto st0
	st19:
		if p++; p == pe {
			goto _test_eof19
		}
	st_case_19:
		if data[p] == 115 {
			goto st20
		}
		goto st0
	st20:
		if p++; p == pe {
			goto _test_eof20
		}
	st_case_20:
		if data[p] == 109 {
			goto st21
		}
		goto st0
	st21:
		if p++; p == pe {
			goto _test_eof21
		}
	st_case_21:
		if data[p] == 116 {
			goto st22
		}
		goto st0
	st22:
		if p++; p == pe {
			goto _test_eof22
		}
	st_case_22:
		if data[p] == 112 {
			goto st23
		}
		goto st0
	st23:
		if p++; p == pe {
			goto _test_eof23
		}
	st_case_23:
		if data[p] == 58 {
			goto st24
		}
		goto st0
	st24:
		if p++; p == pe {
			goto _test_eof24
		}
	st_case_24:
		if data[p] == 91 {
			goto st25
		}
		goto st0
	st25:
		if p++; p == pe {
			goto _test_eof25
		}
	st_case_25:
		if data[p] == 117 {
			goto tr28
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto tr27
		}
		goto st0
tr27:
//line common.rl:29
 tokBeg = p 
	goto st26
	st26:
		if p++; p == pe {
			goto _test_eof26
		}
	st_case_26:
//line smtp.gen.go:2499
		if data[p] == 46 {
			goto st27
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st26
		}
		goto st0
	st27:
		if p++; p == pe {
			goto _test_eof27
		}
	st_case_27:
		if 48 <= data[p] && data[p] <= 57 {
			goto st28
		}
		goto st0
	st28:
		if p++; p == pe {
			goto _test_eof28
		}
	st_case_28:
		if data[p] == 46 {
			goto st29
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st28
		}
		goto st0
	st29:
		if p++; p == pe {
			goto _test_eof29
		}
	st_case_29:
		if 48 <= data[p] && data[p] <= 57 {
			goto st30
		}
		goto st0
	st30:
		if p++; p == pe {
			goto _test_eof30
		}
	st_case_30:
		if data[p] == 46 {
			goto st31
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st30
		}
		goto st0
	st31:
		if p++; p == pe {
			goto _test_eof31
		}
	st_case_31:
		if 48 <= data[p] && data[p] <= 57 {
			goto st32
		}
		goto st0
	st32:
		if p++; p == pe {
			goto _test_eof32
		}
	st_case_32:
		if data[p] == 93 {
			goto tr36
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st32
		}
		goto st0
tr36:
//line smtp.rl:129

		r.IP = data[tokBeg:p]
	
	goto st33
	st33:
		if p++; p == pe {
			goto _test_eof33
		}
	st_case_33:
//line smtp.gen.go:2581
		if data[p] == 58 {
			goto st34
		}
		goto st0
	st34:
		if p++; p == pe {
			goto _test_eof34
		}
	st_case_34:
		if 48 <= data[p] && data[p] <= 57 {
			goto tr38
		}
		goto st0
tr38:
//line common.rl:29
 tokBeg = p 
	goto st35
	st35:
		if p++; p == pe {
			goto _test_eof35
		}
	st_case_35:
//line smtp.gen.go:2604
		if data[p] == 41 {
			goto tr39
		}
		if 48 <= data[p] && data[p] <= 57 {
			goto st35
		}
		goto st0
tr39:
//line smtp.rl:133

		r.Port = data[tokBeg:p]
	
	goto st36
	st36:
		if p++; p == pe {
			goto _test_eof36
		}
	st_case_36:
//line smtp.gen.go:2623
		if data[p] == 58 {
			goto st37
		}
		goto st0
	st37:
		if p++; p == pe {
			goto _test_eof37
		}
	st_case_37:
		if data[p] == 32 {
			goto st38
		}
		goto st0
	st38:
		if p++; p == pe {
			goto _test_eof38
		}
	st_case_38:
		if data[p] == 50 {
			goto st39
		}
		goto st0
	st39:
		if p++; p == pe {
			goto _test_eof39
		}
	st_case_39:
		if data[p] == 53 {
			goto st40
		}
		goto st0
	st40:
		if p++; p == pe {
			goto _test_eof40
		}
	st_case_40:
		if data[p] == 48 {
			goto st41
		}
		goto st0
	st41:
		if p++; p == pe {
			goto _test_eof41
		}
	st_case_41:
		if data[p] == 32 {
			goto st42
		}
		goto st0
	st42:
		if p++; p == pe {
			goto _test_eof42
		}
	st_case_42:
		if data[p] == 50 {
			goto st43
		}
		goto st0
	st43:
		if p++; p == pe {
			goto _test_eof43
		}
	st_case_43:
		if data[p] == 46 {
			goto st44
		}
		goto st0
	st44:
		if p++; p == pe {
			goto _test_eof44
		}
	st_case_44:
		if data[p] == 48 {
			goto st45
		}
		goto st0
	st45:
		if p++; p == pe {
			goto _test_eof45
		}
	st_case_45:
		if data[p] == 46 {
			goto st46
		}
		goto st0
	st46:
		if p++; p == pe {
			goto _test_eof46
		}
	st_case_46:
		if data[p] == 48 {
			goto st47
		}
		goto st0
	st47:
		if p++; p == pe {
			goto _test_eof47
		}
	st_case_47:
		if data[p] == 32 {
			goto st48
		}
		goto st0
	st48:
		if p++; p == pe {
			goto _test_eof48
		}
	st_case_48:
		if data[p] == 79 {
			goto st49
		}
		goto st0
	st49:
		if p++; p == pe {
			goto _test_eof49
		}
	st_case_49:
		if data[p] == 107 {
			goto st50
		}
		goto st0
	st50:
		if p++; p == pe {
			goto _test_eof50
		}
	st_case_50:
		if data[p] == 58 {
			goto st51
		}
		goto st0
	st51:
		if p++; p == pe {
			goto _test_eof51
		}
	st_case_51:
		if data[p] == 32 {
			goto st52
		}
		goto st0
	st52:
		if p++; p == pe {
			goto _test_eof52
		}
	st_case_52:
		if data[p] == 113 {
			goto st53
		}
		goto st0
	st53:
		if p++; p == pe {
			goto _test_eof53
		}
	st_case_53:
		if data[p] == 117 {
			goto st54
		}
		goto st0
	st54:
		if p++; p == pe {
			goto _test_eof54
		}
	st_case_54:
		if data[p] == 101 {
			goto st55
		}
		goto st0
	st55:
		if p++; p == pe {
			goto _test_eof55
		}
	st_case_55:
		if data[p] == 117 {
			goto st56
		}
		goto st0
	st56:
		if p++; p == pe {
			goto _test_eof56
		}
	st_case_56:
		if data[p] == 101 {
			goto st57
		}
		goto st0
	st57:
		if p++; p == pe {
			goto _test_eof57
		}
	st_case_57:
		if data[p] == 100 {
			goto st58
		}
		goto st0
	st58:
		if p++; p == pe {
			goto _test_eof58
		}
	st_case_58:
		if data[p] == 32 {
			goto st59
		}
		goto st0
	st59:
		if p++; p == pe {
			goto _test_eof59
		}
	st_case_59:
		if data[p] == 97 {
			goto st60
		}
		goto st0
	st60:
		if p++; p == pe {
			goto _test_eof60
		}
	st_case_60:
		if data[p] == 115 {
			goto st61
		}
		goto st0
	st61:
		if p++; p == pe {
			goto _test_eof61
		}
	st_case_61:
		if data[p] == 32 {
			goto st62
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
				goto tr67
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto tr68
				}
			case data[p] >= 71:
				goto tr68
			}
		default:
			goto tr67
		}
		goto st0
tr67:
//line common.rl:29
 tokBeg = p 
	goto st63
	st63:
		if p++; p == pe {
			goto _test_eof63
		}
	st_case_63:
//line smtp.gen.go:2885
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st64
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st83
				}
			case data[p] >= 71:
				goto st83
			}
		default:
			goto st64
		}
		goto st0
	st64:
		if p++; p == pe {
			goto _test_eof64
		}
	st_case_64:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st65
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st82
				}
			case data[p] >= 71:
				goto st82
			}
		default:
			goto st65
		}
		goto st0
	st65:
		if p++; p == pe {
			goto _test_eof65
		}
	st_case_65:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st66
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st81
				}
			case data[p] >= 71:
				goto st81
			}
		default:
			goto st66
		}
		goto st0
	st66:
		if p++; p == pe {
			goto _test_eof66
		}
	st_case_66:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st67
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st80
				}
			case data[p] >= 71:
				goto st80
			}
		default:
			goto st67
		}
		goto st0
	st67:
		if p++; p == pe {
			goto _test_eof67
		}
	st_case_67:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st68
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st79
				}
			case data[p] >= 71:
				goto st79
			}
		default:
			goto st68
		}
		goto st0
	st68:
		if p++; p == pe {
			goto _test_eof68
		}
	st_case_68:
		if data[p] == 41 {
			goto tr79
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st69
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st78
				}
			case data[p] >= 71:
				goto st78
			}
		default:
			goto st69
		}
		goto st0
tr79:
//line smtp.rl:137

		r.Queue = data[tokBeg:p]
	
	goto st92
	st92:
		if p++; p == pe {
			goto _test_eof92
		}
	st_case_92:
//line smtp.gen.go:3033
		goto st0
	st69:
		if p++; p == pe {
			goto _test_eof69
		}
	st_case_69:
		if data[p] == 41 {
			goto tr79
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st70
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st77
				}
			case data[p] >= 71:
				goto st77
			}
		default:
			goto st70
		}
		goto st0
	st70:
		if p++; p == pe {
			goto _test_eof70
		}
	st_case_70:
		if data[p] == 41 {
			goto tr79
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st71
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st76
				}
			case data[p] >= 71:
				goto st76
			}
		default:
			goto st71
		}
		goto st0
	st71:
		if p++; p == pe {
			goto _test_eof71
		}
	st_case_71:
		if data[p] == 41 {
			goto tr79
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st72
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st75
				}
			case data[p] >= 71:
				goto st75
			}
		default:
			goto st72
		}
		goto st0
	st72:
		if p++; p == pe {
			goto _test_eof72
		}
	st_case_72:
		if data[p] == 41 {
			goto tr79
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st73
			}
		case data[p] > 70:
			switch {
			case data[p] > 90:
				if 97 <= data[p] && data[p] <= 122 {
					goto st74
				}
			case data[p] >= 71:
				goto st74
			}
		default:
			goto st73
		}
		goto st0
	st73:
		if p++; p == pe {
			goto _test_eof73
		}
	st_case_73:
		if data[p] == 41 {
			goto tr79
		}
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st73
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st73
			}
		default:
			goto st73
		}
		goto st0
	st74:
		if p++; p == pe {
			goto _test_eof74
		}
	st_case_74:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st73
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st73
			}
		default:
			goto st73
		}
		goto st0
	st75:
		if p++; p == pe {
			goto _test_eof75
		}
	st_case_75:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st74
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st74
			}
		default:
			goto st74
		}
		goto st0
	st76:
		if p++; p == pe {
			goto _test_eof76
		}
	st_case_76:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st75
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st75
			}
		default:
			goto st75
		}
		goto st0
	st77:
		if p++; p == pe {
			goto _test_eof77
		}
	st_case_77:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st76
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st76
			}
		default:
			goto st76
		}
		goto st0
	st78:
		if p++; p == pe {
			goto _test_eof78
		}
	st_case_78:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st77
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st77
			}
		default:
			goto st77
		}
		goto st0
	st79:
		if p++; p == pe {
			goto _test_eof79
		}
	st_case_79:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st78
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st78
			}
		default:
			goto st78
		}
		goto st0
	st80:
		if p++; p == pe {
			goto _test_eof80
		}
	st_case_80:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st79
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st79
			}
		default:
			goto st79
		}
		goto st0
	st81:
		if p++; p == pe {
			goto _test_eof81
		}
	st_case_81:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st80
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st80
			}
		default:
			goto st80
		}
		goto st0
	st82:
		if p++; p == pe {
			goto _test_eof82
		}
	st_case_82:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st81
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st81
			}
		default:
			goto st81
		}
		goto st0
	st83:
		if p++; p == pe {
			goto _test_eof83
		}
	st_case_83:
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st82
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st82
			}
		default:
			goto st82
		}
		goto st0
tr68:
//line common.rl:29
 tokBeg = p 
	goto st84
	st84:
		if p++; p == pe {
			goto _test_eof84
		}
	st_case_84:
//line smtp.gen.go:3349
		switch {
		case data[p] < 65:
			if 48 <= data[p] && data[p] <= 57 {
				goto st83
			}
		case data[p] > 90:
			if 97 <= data[p] && data[p] <= 122 {
				goto st83
			}
		default:
			goto st83
		}
		goto st0
tr28:
//line common.rl:29
 tokBeg = p 
	goto st85
	st85:
		if p++; p == pe {
			goto _test_eof85
		}
	st_case_85:
//line smtp.gen.go:3372
		if data[p] == 110 {
			goto st86
		}
		goto st0
	st86:
		if p++; p == pe {
			goto _test_eof86
		}
	st_case_86:
		if data[p] == 107 {
			goto st87
		}
		goto st0
	st87:
		if p++; p == pe {
			goto _test_eof87
		}
	st_case_87:
		if data[p] == 110 {
			goto st88
		}
		goto st0
	st88:
		if p++; p == pe {
			goto _test_eof88
		}
	st_case_88:
		if data[p] == 111 {
			goto st89
		}
		goto st0
	st89:
		if p++; p == pe {
			goto _test_eof89
		}
	st_case_89:
		if data[p] == 119 {
			goto st90
		}
		goto st0
	st90:
		if p++; p == pe {
			goto _test_eof90
		}
	st_case_90:
		if data[p] == 110 {
			goto st91
		}
		goto st0
	st91:
		if p++; p == pe {
			goto _test_eof91
		}
	st_case_91:
		if data[p] == 93 {
			goto tr36
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
	_test_eof41: cs = 41; goto _test_eof
	_test_eof42: cs = 42; goto _test_eof
	_test_eof43: cs = 43; goto _test_eof
	_test_eof44: cs = 44; goto _test_eof
	_test_eof45: cs = 45; goto _test_eof
	_test_eof46: cs = 46; goto _test_eof
	_test_eof47: cs = 47; goto _test_eof
	_test_eof48: cs = 48; goto _test_eof
	_test_eof49: cs = 49; goto _test_eof
	_test_eof50: cs = 50; goto _test_eof
	_test_eof51: cs = 51; goto _test_eof
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
	_test_eof65: cs = 65; goto _test_eof
	_test_eof66: cs = 66; goto _test_eof
	_test_eof67: cs = 67; goto _test_eof
	_test_eof68: cs = 68; goto _test_eof
	_test_eof92: cs = 92; goto _test_eof
	_test_eof69: cs = 69; goto _test_eof
	_test_eof70: cs = 70; goto _test_eof
	_test_eof71: cs = 71; goto _test_eof
	_test_eof72: cs = 72; goto _test_eof
	_test_eof73: cs = 73; goto _test_eof
	_test_eof74: cs = 74; goto _test_eof
	_test_eof75: cs = 75; goto _test_eof
	_test_eof76: cs = 76; goto _test_eof
	_test_eof77: cs = 77; goto _test_eof
	_test_eof78: cs = 78; goto _test_eof
	_test_eof79: cs = 79; goto _test_eof
	_test_eof80: cs = 80; goto _test_eof
	_test_eof81: cs = 81; goto _test_eof
	_test_eof82: cs = 82; goto _test_eof
	_test_eof83: cs = 83; goto _test_eof
	_test_eof84: cs = 84; goto _test_eof
	_test_eof85: cs = 85; goto _test_eof
	_test_eof86: cs = 86; goto _test_eof
	_test_eof87: cs = 87; goto _test_eof
	_test_eof88: cs = 88; goto _test_eof
	_test_eof89: cs = 89; goto _test_eof
	_test_eof90: cs = 90; goto _test_eof
	_test_eof91: cs = 91; goto _test_eof

	_test_eof: {}
	if p == eof {
		switch cs {
		case 92:
//line smtp.rl:143

		return r, true
	
//line smtp.gen.go:3532
		}
	}

	_out: {}
	}

//line smtp.rl:149


	return r, false
}
