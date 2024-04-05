package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"

	config "kaoconfig"
	databasepool "kaodatabasepool"

	//"kaoreqreceive"

	//"kaocenter"
	"kaosendrequest"
	"oshotproc"
	"otpproc"
	//"strconv"
	//"time"
	s "strings"
	//"github.com/gin-gonic/gin"
	"github.com/takama/daemon"
)

const (
	name        = "DHNNanoServer"
	description = "대형네트웍스 카카오 발송 서버"
)

var dependencies = []string{"DHNNanoServer.service"}

var resultTable string

type Service struct {
	daemon.Daemon
}

func (service *Service) Manage() (string, error) {

	usage := "Usage: DHNNanoServer install | remove | start | stop | status"

	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return usage, nil
		}
	}
	resultProc()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, os.Kill, syscall.SIGTERM)

	for {
		select {
		case killSignal := <-interrupt:
			config.Stdlog.Println("Got signal:", killSignal)
			config.Stdlog.Println("Stoping DB Conntion : ", databasepool.DB.Stats())
			defer databasepool.DB.Close()
			if killSignal == os.Interrupt {
				return "Daemon was interrupted by system signal", nil
			}
			return "Daemon was killed", nil
		}
	}
}

func main() {

	config.InitConfig()

	databasepool.InitDatabase()
	
	var rLimit syscall.Rlimit
	
	rLimit.Max = 50000
    rLimit.Cur = 50000
    
    err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    
    if err != nil {
        config.Stdlog.Println("Error Setting Rlimit ", err)
    }
    
    err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
    
    if err != nil {
        config.Stdlog.Println("Error Getting Rlimit ", err)
    }
    
    config.Stdlog.Println("Rlimit Final", rLimit)
    
	srv, err := daemon.New(name, description, daemon.SystemDaemon, dependencies...)
	if err != nil {
		config.Stdlog.Println("Error: ", err)
		os.Exit(1)
	}
	service := &Service{srv}
	status, err := service.Manage()
	if err != nil {
		config.Stdlog.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	fmt.Println(status)
}

func resultProc() {
	config.Stdlog.Println("DHN Server 시작")

	go kaosendrequest.AlimtalkProc()

	go kaosendrequest.FriendtalkProc()

	if  s.EqualFold(config.Conf.RESPONSE_METHOD, "polling") {
		go kaosendrequest.PollingProc()
	}

	go kaosendrequest.ResultProc()
	
	if s.EqualFold(config.Conf.PHONE_MSG_FLAG, "YES") {
		go oshotproc.OshotProcess()
	
		go oshotproc.LMSProcess()
		
		go oshotproc.SMSProcess()
	}
	
	if s.EqualFold(config.Conf.OTP_MSG_FLAG, "YES") {
		go otpproc.OTPProcess()
	
		go otpproc.OTPLMSProcess()
		
		go otpproc.OTPSMSProcess()
	}
	
	/*
		r := gin.New()
		r.Use(gin.Recovery())
		//r := gin.Default()

		r.GET("/", func(c *gin.Context) {
			c.String(200, "Server : "+config.Conf.CENTER_SERVER)
		})

		r.POST("/req", kaoreqreceive.ReqReceive)

		r.POST("/result", kaoreqreceive.Resultreq)

		r.GET("/sender/token", kaocenter.Sender_token)

		r.GET("/category/all", kaocenter.Category_all)

		r.GET("/sender/category", kaocenter.Category_)

		r.POST("/sender/create", kaocenter.Sender_Create)

		r.GET("/sender", kaocenter.Sender_)

		r.POST("/sender/delete", kaocenter.Sender_Delete)

		r.POST("/sender/recover", kaocenter.Sender_Recover)

		r.POST("/template/create", kaocenter.Template_Create)

		r.POST("/template/create_with_image", kaocenter.Template_Create_Image)

		r.GET("/template", kaocenter.Template_)

		r.POST("/template/request", kaocenter.Template_Request)

		r.POST("/template/cancel_request", kaocenter.Template_Cancel_Request)

		r.POST("/template/update", kaocenter.Template_Update)

		r.POST("/template/update_with_image", kaocenter.Template_Update_Image)

		r.POST("/template/stop", kaocenter.Template_Stop)

		r.POST("/template/reuse", kaocenter.Template_Reuse)

		r.POST("/template/delete", kaocenter.Template_Delete)

		r.GET("/template/last_modified", kaocenter.Template_Last_Modified)

		r.POST("/template/comment", kaocenter.Template_Comment)

		r.POST("/template/comment_file", kaocenter.Template_Comment_File)

		r.GET("/template/category/all", kaocenter.Template_Category_all)

		r.GET("/template/category", kaocenter.Template_Category_)

		r.POST("/template/category/update", kaocenter.Template_Category_Update)

		r.POST("/template/dormant/release", kaocenter.Template_Dormant_Release)

		r.GET("/group", kaocenter.Group_)

		r.GET("/group/sender", kaocenter.Group_Sender)

		r.POST("/group/sender/add", kaocenter.Group_Sender_Add)

		r.POST("/group/sender/remove", kaocenter.Group_Sender_Remove)

		r.POST("/channel/create", kaocenter.Channel_Create_)

		r.GET("/channel/all", kaocenter.Channel_all)

		r.GET("/channel", kaocenter.Channel_)

		r.POST("/channel/update", kaocenter.Channel_Update_)

		r.POST("/channel/senders", kaocenter.Channel_Senders_)

		r.POST("/channel/delete", kaocenter.Channel_Delete_)

		r.GET("/plugin/callbackUrl/list", kaocenter.Plugin_CallbackUrls_List)

		r.POST("/plugin/callbackUrl/create", kaocenter.Plugin_callbackUrl_Create)

		r.POST("/plugin/callbackUrl/update", kaocenter.Plugin_callbackUrl_Update)

		r.POST("/plugin/callbackUrl/delete", kaocenter.Plugin_callbackUrl_Delete)

		r.POST("/ft/image", kaocenter.FT_Upload)

		r.POST("/ft/wide/image", kaocenter.FT_Wide_Upload)

		r.Run(":" + config.Conf.PORT)
	*/
}
