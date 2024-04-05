package oshotproc

import (
	"database/sql"
	"fmt"
	//"sync"
	config "kaoconfig"
	databasepool "kaodatabasepool"

	"encoding/hex"
	"regexp"
	s "strings"
	"time"
	"unicode/utf8"
	//"bytes"
	//iconv "github.com/djimenez/iconv-go"
	//"golang.org/x/text/encoding/korean"
	//"golang.org/x/text/transform"
)

var procCnt int

func OshotProcess() {
	//var wg sync.WaitGroup
	procCnt = 0
	for {
		if procCnt < 5 {
			var count int

			cnterr := databasepool.DB.QueryRow("select length(msgid) as cnt from DHN_RESULT  where result = 'P' and send_group is null and ifnull(reserve_dt,'00000000000000') <= date_format(now(), '%Y%m%d%H%i%S') limit 1").Scan(&count)

			if cnterr != nil {
				//config.Stdlog.Println("DHN_RESULT Table - select 오류 : " + cnterr.Error())
			} else {

				if count > 0 {

					//wg.Add(1)

					var startNow = time.Now()
					var group_no = fmt.Sprintf("%02d%02d%02d%02d%06d", startNow.Day(), startNow.Hour(), startNow.Minute(), startNow.Second(), (startNow.Nanosecond()/1000))
					
					//config.Stdlog.Println(group_no, " Update 시작")
					//updateRows, err := databasepool.DB.Exec("update DHN_RESULT set send_group = '" + group_no + "' where  result = 'P' and ( length(send_group) <=0 or send_group is null ) and ifnull(reserve_dt,'00000000000000') <= date_format(now(), '%Y%m%d%H%i%S') LIMIT 1000")
					//if err != nil {
					//	config.Stdlog.Println("DHN_RESULT Table - Group No Update 오류" + err.Error())
					//}
					//rowcnt, _ := updateRows.RowsAffected()

					//config.Stdlog.Println(group_no, " Update 끝 ", rowcnt)
					updateReqeust(group_no)
					//if rowcnt > 0 {
					go resProcess(group_no)
					//}
				}
			}
		}
	}
}

func updateReqeust(group_no string) {

	tx, err := databasepool.DB.Begin()
	if err != nil {
		return
	}

	defer func() {
		//config.Stdlog.Println("Group No Update End", group_no)
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()
	
	//tx := databasepool.DB
	//cnt := 0
	
	config.Stdlog.Println("Group No Update 시작", group_no)
	/*	
	reqrows, err := tx.Query("select msgid from DHN_RESULT where  result = 'P' and send_group is null and ifnull(reserve_dt,'00000000000000') <= date_format(now(), '%Y%m%d%H%i%S') LIMIT 500")
	if err != nil {
		config.Stdlog.Println(" Group NO Update - Select 오류 : ( " + group_no + " ) : " + err.Error())
		return
	}
	
	for reqrows.Next() {
		var msgid sql.NullString
		reqrows.Scan(&msgid)
		
		if _, err = tx.Exec("update DHN_RESULT set send_group = '" + group_no + "' where  msgid = '" + msgid.String +"'"); err != nil {
			config.Stdlog.Println("update DHN_RESULT set send_group = '" + group_no + "' where  msgid = '" + msgid.String +"'", err)
			return
		} 
		cnt++
	}
	*/

	gudQuery := "update DHN_RESULT set send_group = '" + group_no + "' where  result = 'P' and send_group is null and ifnull(reserve_dt,'00000000000000') <= date_format(now(), '%Y%m%d%H%i%S') LIMIT 500"
	_, err = tx.Query(gudQuery)

	if err != nil {
		config.Stdlog.Println(" Group NO Update - Select error : ( " + group_no + " ) : " + err.Error())
		config.Stdlog.Println(gudQuery)
		return
	}
	
	return
}

func resProcess(group_no string) {
	//defer wg.Done()

	procCnt++
	var db = databasepool.DB
	var stdlog = config.Stdlog

	defer func() {
		if err := recover(); err != nil {
			procCnt--
			stdlog.Println(group_no, "KAO 처리 중 오류 발생 : ", err)
		}
	} ()
	
	var msgid, code, message, message_type, msg_sms, phn, remark1, remark2, result, sms_lms_tit, sms_kind, sms_sender, res_dt, reserve_dt, mms_file1, mms_file2, mms_file3, userid, sms_len_check, oshot sql.NullString
	var msgLen sql.NullInt64
	var phnstr string

	ossmsStrs := []string{}
	ossmsValues := []interface{}{}

	osmmsStrs := []string{}
	osmmsValues := []interface{}{}

	var resquery = `SELECT msgid, 
	code, 
	message, 
	message_type, 
	(case when sms_kind = 'S' then 
		substr(convert(REMOVE_WS(msg_sms) using euckr),1,100)
	 else 
	   convert(REMOVE_WS(msg_sms) using euckr)
     end) as msg_sms, 
	phn, 
	remark1, 
	remark2,
	result, 
	convert(REMOVE_WS(sms_lms_tit) using euckr) as sms_lms_tit, 
	sms_kind, 
	sms_sender, 
	res_dt, 
	reserve_dt, 
	(select file1_path from api_mms_images aa where aa.user_id = drr.userid and aa.mms_id = drr.p_invoice) as mms_file1, 
	(select file2_path from api_mms_images aa where aa.user_id = drr.userid and aa.mms_id = drr.p_invoice) as mms_file2, 
	(select file3_path from api_mms_images aa where aa.user_id = drr.userid and aa.mms_id = drr.p_invoice) as mms_file3
	,(case when sms_kind = 'S' then length(convert(REMOVE_WS(msg_sms) using euckr)) else 100 end) as msg_len
	,userid
	,(select max(sms_len_check) from DHN_CLIENT_LIST dcl where dcl.user_id = drr.userid) as sms_len_check
	,(select ifnull(max(oshot), 'OShot') from DHN_CLIENT_LIST dcl where dcl.user_id = drr.userid) as oshot  
	FROM DHN_RESULT drr 
	WHERE send_group = '` + group_no + `' 
	order by userid
	`

	resrows, err := db.Query(resquery)

	if err != nil {
		stdlog.Println("Result Table 조회 중 오류 발생")
		stdlog.Println(err)
		stdlog.Println(resquery)
	}
	defer resrows.Close()
	scnt := 0
	fcnt := 0
	smscnt := 0
	lmscnt := 0
	tcnt := 0
	reg, err := regexp.Compile("[^0-9]+")
	preOshot := ""

	for resrows.Next() {
		resrows.Scan(&msgid, &code, &message, &message_type, &msg_sms, &phn, &remark1, &remark2, &result, &sms_lms_tit, &sms_kind, &sms_sender, &res_dt, &reserve_dt, &mms_file1, &mms_file2, &mms_file3, &msgLen, &userid, &sms_len_check, &oshot)

		phnstr = phn.String

		if tcnt == 0 {
			stdlog.Println(group_no, "문자발송 처리 시작 : ", " Process cnt : ", procCnt)
			preOshot = oshot.String
		}

		tcnt++


		if len(ossmsStrs) > 500 || ( preOshot != oshot.String && len(ossmsStrs) > 0 ) {
			stmt := fmt.Sprintf("insert into " + preOshot + "SMS(Sender,Receiver,Msg,URL,ReserveDT,TimeoutDT,SendResult,mst_id,cb_msg_id,userid ) values %s", s.Join(ossmsStrs, ","))
			_, err := db.Exec(stmt, ossmsValues...)

			if err != nil {
				//stdlog.Println("스마트미 SMS Table Insert 처리 중 오류 발생 " + err.Error())
				for i := 0; i < len(ossmsValues); i = i + 9 {
					eQuery := fmt.Sprintf("insert into " + preOshot + "SMS(Sender,Receiver,Msg,URL,ReserveDT,TimeoutDT,SendResult,mst_id,cb_msg_id,userid ) "+
						"values('%v','%v','%v','%v',null,null,'%v',null,'%v','%v')", ossmsValues[i], ossmsValues[i+1], ossmsValues[i+2], ossmsValues[i+3], ossmsValues[i+5], ossmsValues[i+7],ossmsValues[i+8])
					_, err := db.Exec(eQuery)
					if err != nil {
						msgKey := fmt.Sprintf("%v", ossmsValues[i+7])
						useridt := fmt.Sprintf("%v", ossmsValues[i+8])
						stdlog.Println("스마트미 SMS Table Insert 처리 중 오류 발생 : "+err.Error(), " - DHN Msg Key : ", msgKey)
						errcodemsg := err.Error()
						if s.Index(errcodemsg, "1366") > 0 {
							db.Exec("update DHN_RESULT dr set dr.result = 'Y', dr.code='7069', dr.message = concat(dr.message, ',부적절한 문자사용'),dr.remark2 = date_format(now(), '%Y-%m-%d %H:%i:%S') where userid = '"+ useridt +"' and  msgid = '" + msgKey + "'")
						} 
					}
				}
				//db.Exec("update API_RESULT ar set ar.msg_type = '" + sms_kind.String + "', result_code = '9999', error_text = '기타오류', report_time = date_format(now(), '%Y-%m-%d %H:%i:%S') where dhn_msg_id = '" + msgid.String + "'")
			} else {
				stdlog.Println("스마트미 SMS Table Insert 처리 : ", len(ossmsStrs), " - ", preOshot)
			}
			ossmsStrs = nil
			ossmsValues = nil
		}

		if len(osmmsStrs) > 500 || ( preOshot != oshot.String && len(osmmsStrs) > 0 ) {
			stmt := fmt.Sprintf("insert into " + preOshot + "MMS(MsgGroupID,Sender,Receiver,Subject,Msg,ReserveDT,TimeoutDT,SendResult,File_Path1,File_Path2,File_Path3,mst_id,cb_msg_id,userid ) values %s", s.Join(osmmsStrs, ","))
			_, err := db.Exec(stmt, osmmsValues...)

			if err != nil {
				//stdlog.Println("스마트미 SMS Table Insert 처리 중 오류 발생 " + err.Error())
				for i := 0; i < len(osmmsValues); i = i + 13 {
					eQuery := fmt.Sprintf("insert into " + preOshot + "MMS(MsgGroupID,Sender,Receiver,Subject,Msg,ReserveDT,TimeoutDT,SendResult,File_Path1,File_Path2,File_Path3,mst_id,cb_msg_id,userid ) "+
						"values('%v','%v','%v','%v','%v',null,null,'%v','%v','%v','%v',null,'%v','%v')", osmmsValues[i], osmmsValues[i+1], osmmsValues[i+2], osmmsValues[i+3], osmmsValues[i+4], osmmsValues[i+6], osmmsValues[i+7], osmmsValues[i+8], osmmsValues[i+9], osmmsValues[i+11], osmmsValues[i+12])
					_, err := db.Exec(eQuery)
					if err != nil {
						msgKey := fmt.Sprintf("%v", osmmsValues[i+11])
						useridt := fmt.Sprintf("%v", osmmsValues[i+12])
						stdlog.Println("스마트미 LMS Table Insert 처리 중 오류 발생 : "+err.Error(), " - DHN Msg Key : ", msgKey)
						errcodemsg := err.Error()
						if s.Index(errcodemsg, "1366") > 0 {
							db.Exec("update DHN_RESULT dr set dr.result = 'Y', dr.code='7069', dr.message = concat(dr.message, ',부적절한 문자사용'),dr.remark2 = date_format(now(), '%Y-%m-%d %H:%i:%S') where userid = '"+ useridt +"' and msgid = '" + msgKey + "'")
						} 
					}
				}
				//db.Exec("update API_RESULT ar set ar.msg_type = '" + sms_kind.String + "', result_code = '9999', error_text = '기타오류', report_time = date_format(now(), '%Y-%m-%d %H:%i:%S') where dhn_msg_id = '" + msgid.String + "'")
			}else {
				stdlog.Println("스마트미 MMS Table Insert 처리 : ", len(osmmsStrs), " - ", preOshot)
			}
			osmmsStrs = nil
			osmmsValues = nil
		}
		
		// 알림톡 발송 성공 혹은 문자 발송이 아니면
		// API_RESULT 성공 처리 함.
		if len(msg_sms.String) > 0 && len(sms_sender.String) > 0 { // msg_sms 가 와 sms_sender 에 값이 있으면 Oshot 발송 함.
		 
			phnstr = reg.ReplaceAllString(phnstr, "")
			if s.HasPrefix(phnstr, "82") {
				phnstr = "0"+phnstr[2:len(phnstr)]
			}

			if s.EqualFold(sms_kind.String, "S") {
				//smsmsg := utf8TOeuckr(stringSplit(msg_sms.String, 100))
				if msgLen.Int64 <= 90 || s.EqualFold(sms_len_check.String, "N") {
					//smsmsg := utf8TOeuckr(stringSplit(msg_sms.String, 100))
					ossmsStrs = append(ossmsStrs, "(?,?,?,?,?,null,?,?,?,?)")
					ossmsValues = append(ossmsValues, sms_sender.String)
					ossmsValues = append(ossmsValues, phnstr)
					ossmsValues = append(ossmsValues, msg_sms.String)
					ossmsValues = append(ossmsValues, "")
					if s.EqualFold(reserve_dt.String, "00000000000000") {
						ossmsValues = append(ossmsValues, sql.NullString{})
					} else {
						ossmsValues = append(ossmsValues, sql.NullString{})
					}
					ossmsValues = append(ossmsValues, "0")
					ossmsValues = append(ossmsValues, sql.NullString{})
					ossmsValues = append(ossmsValues, msgid.String)
					ossmsValues = append(ossmsValues, userid.String)
					smscnt++
				} else {
					db.Exec("update DHN_RESULT dr set dr.result = 'Y', dr.code = '7003', dr.message = '메세지 길이 오류', dr.remark2 = date_format(now(), '%Y-%m-%d %H:%i:%S') where userid = '" + userid.String + "' and msgid = '" + msgid.String + "'")				
				}
			} else if s.EqualFold(sms_kind.String, "L") || s.EqualFold(sms_kind.String, "M") {
				//stdlog.Println(msg_sms.String)
				//lmsmsg := utf8TOeuckr(msg_sms.String)
				//lmsmsg := msg_sms.String
				//stdlog.Println(lmsmsg)
				//lmstmsg := utf8TOeuckr(sms_lms_tit.String)
				osmmsStrs = append(osmmsStrs, "(?,?,?,?,?,?,null,?,?,?,?,?,?,?)")
				/*if len(mms_file1.String) > 0 {
					osmmsValues = append(osmmsValues, remark2.String)
				} else {
					osmmsValues = append(osmmsValues, remark1.String)
				}
				*/
				osmmsValues = append(osmmsValues, group_no)
				osmmsValues = append(osmmsValues, sms_sender.String)
				osmmsValues = append(osmmsValues, phnstr)
				osmmsValues = append(osmmsValues, sms_lms_tit.String)
				osmmsValues = append(osmmsValues, msg_sms.String)
				if s.EqualFold(reserve_dt.String, "00000000000000") {
					osmmsValues = append(osmmsValues, sql.NullString{})
				} else {
					osmmsValues = append(osmmsValues, sql.NullString{})
				}
				osmmsValues = append(osmmsValues, "0")
				osmmsValues = append(osmmsValues, mms_file1.String)
				osmmsValues = append(osmmsValues, mms_file2.String)
				osmmsValues = append(osmmsValues, mms_file3.String)
				osmmsValues = append(osmmsValues, sql.NullString{})
				osmmsValues = append(osmmsValues, msgid.String)
				osmmsValues = append(osmmsValues, userid.String)
				lmscnt++
			}
	 

			preOshot = oshot.String
		} else {
			db.Exec("update DHN_RESULT dr set dr.result = 'Y', dr.code='7011', dr.message = concat(dr.message, ',문자 발송 정보 누락'),dr.remark2 = date_format(now(), '%Y-%m-%d %H:%i:%S') where userid = '"+ userid.String +"' and msgid = '" + msgid.String + "'")
		}
		
	}

	if len(ossmsStrs) > 0 {
		stmt := fmt.Sprintf("insert into " + preOshot + "SMS(Sender,Receiver,Msg,URL,ReserveDT,TimeoutDT,SendResult,mst_id,cb_msg_id,userid) values %s", s.Join(ossmsStrs, ","))
		_, err := db.Exec(stmt, ossmsValues...)

		if err != nil {
			stdlog.Println("스마트미 SMS Table Insert 처리 중 오류 발생 " + err.Error())
			for i := 0; i < len(ossmsValues); i = i + 9 {
				eQuery := fmt.Sprintf("insert into " + preOshot + "SMS(Sender,Receiver,Msg,URL,ReserveDT,TimeoutDT,SendResult,mst_id,cb_msg_id,userid ) "+
					"values('%v','%v','%v','%v','%v',null,'%v','%v','%v','%v')", ossmsValues[i], ossmsValues[i+1], ossmsValues[i+2], ossmsValues[i+3], ossmsValues[i+4], ossmsValues[i+5], ossmsValues[i+6], ossmsValues[i+7], ossmsValues[i+8])
				_, err := db.Exec(eQuery)
				if err != nil {
					msgKey := fmt.Sprintf("%v", ossmsValues[i+7])
					useridt := fmt.Sprintf("%v", ossmsValues[i+8])
					stdlog.Println("스마트미 SMS Table Insert 처리 중 오류 발생 : "+err.Error(), " - DHN Msg Key : ", msgKey)
					errcodemsg := err.Error()
					if s.Index(errcodemsg, "1366") > 0 {
						db.Exec("update DHN_RESULT dr set dr.result = 'Y', dr.code='7069', dr.message = concat(dr.message, ',부적절한 문자사용'),dr.remark2 = date_format(now(), '%Y-%m-%d %H:%i:%S') where userid = '"+ useridt +"' and msgid = '" + msgKey + "'")
					} 
				}
			}
			//db.Exec("update API_RESULT ar set ar.msg_type = '" + sms_kind.String + "', result_code = '9999', error_text = '기타오류', report_time = date_format(now(), '%Y-%m-%d %H:%i:%S') where dhn_msg_id = '" + msgid.String + "'")
		} else {
			stdlog.Println("스마트미 SMS Table Insert 처리 : ", len(ossmsStrs), " - ", preOshot)
		}

	}

	if len(osmmsStrs) > 0 {
		stmt := fmt.Sprintf("insert into " + preOshot + "MMS(MsgGroupID,Sender,Receiver,Subject,Msg,ReserveDT,TimeoutDT,SendResult,File_Path1,File_Path2,File_Path3,mst_id,cb_msg_id,userid ) values %s", s.Join(osmmsStrs, ","))
		_, err := db.Exec(stmt, osmmsValues...)

		if err != nil {
			stdlog.Println("스마트미 SMS Table Insert 처리 중 오류 발생 " + err.Error())
			for i := 0; i < len(osmmsValues); i = i + 13 {
				eQuery := fmt.Sprintf("insert into " + preOshot + "MMS(MsgGroupID,Sender,Receiver,Subject,Msg,ReserveDT,TimeoutDT,SendResult,File_Path1,File_Path2,File_Path3,mst_id,cb_msg_id,userid ) "+
					"values('%v','%v','%v','%v','%v',null,null,'%v','%v','%v','%v',null,'%v','%v')", osmmsValues[i], osmmsValues[i+1], osmmsValues[i+2], osmmsValues[i+3], osmmsValues[i+4], osmmsValues[i+6], osmmsValues[i+7], osmmsValues[i+8], osmmsValues[i+9], osmmsValues[i+11], osmmsValues[i+12])
				_, err := db.Exec(eQuery)
				if err != nil {
					msgKey := fmt.Sprintf("%v", osmmsValues[i+11])
					useridt := fmt.Sprintf("%v", osmmsValues[i+12])
					stdlog.Println("스마트미 LMS Table Insert 처리 중 오류 발생 : "+err.Error(), " - DHN Msg Key : ", msgKey)
					errcodemsg := err.Error()
					if s.Index(errcodemsg, "1366") > 0 {
						db.Exec("update DHN_RESULT dr set dr.result = 'Y', dr.code='7069', dr.message = concat(dr.message, ',부적절한 문자사용'),dr.remark2 = date_format(now(), '%Y-%m-%d %H:%i:%S') where userid = '"+ useridt +"' and msgid = '" + msgKey + "'")
					} 
				}
			}
			//db.Exec("update API_RESULT ar set ar.msg_type = '" + sms_kind.String + "', result_code = '9999', error_text = '기타오류', report_time = date_format(now(), '%Y-%m-%d %H:%i:%S') where dhn_msg_id = '" + msgid.String + "'")
		} else {
			stdlog.Println("스마트미 MMS Table Insert 처리 : ", len(osmmsStrs), " - ", preOshot)
		}

	}

	if scnt > 0 || smscnt > 0 || lmscnt > 0 || fcnt > 0 {
		stdlog.Println(group_no, "문자 발송 처리 완료 ( ", tcnt, " ) : 성공 -", scnt, " , SMS -", smscnt, " , LMS -", lmscnt, "실패 - ", fcnt, "  >> Process cnt : ", procCnt)
	}
	procCnt--
}

func stringSplit(str string, lencnt int) string {
	b := []byte(str)
	idx := 0
	for i := 0; i < lencnt; i++ {
		_, size := utf8.DecodeRune(b[idx:])
		idx += size
	}
	return str[:idx]
}

func utf8TOeuckr(str string) string {
	sText := []byte(str)
	eText := make([]byte, hex.EncodedLen(len(sText)))
	hex.Encode(eText, sText)

	temp := string(eText)
	temp = s.Replace(temp, "e2808b", "", -1)

	bs, _ := hex.DecodeString(temp)

	return string(bs)
}
