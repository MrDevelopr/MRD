package job

import (
	"fmt"
	"net"
	"os"
	"time"
	"x-ui/logger"
	"x-ui/util/common"
	"x-ui/web/service"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type LoginStatus byte

const (
	LoginSuccess LoginStatus = 1
	LoginFail    LoginStatus = 0
)

type StatsNotifyJob struct {
	enable         bool
	xrayService    service.XrayService
	inboundService service.InboundService
	settingService service.SettingService
}

func NewStatsNotifyJob() *StatsNotifyJob {
	return new(StatsNotifyJob)
}

func (j *StatsNotifyJob) SendMsgToTgbot(msg string) {
	//Telegram bot basic info
	tgBottoken, err := j.settingService.GetTgBotToken()
	if err != nil || tgBottoken == "" {
		logger.Warning("sendMsgToTgbot failed,GetTgBotToken fail:", err)
		return
	}
	tgBotid, err := j.settingService.GetTgBotChatId()
	if err != nil {
		logger.Warning("sendMsgToTgbot failed,GetTgBotChatId fail:", err)
		return
	}

	bot, err := tgbotapi.NewBotAPI(tgBottoken)
	if err != nil {
		fmt.Println("get tgbot error:", err)
		return
	}
	bot.Debug = true
	fmt.Printf("Authorized on account %s", bot.Self.UserName)
	info := tgbotapi.NewMessage(int64(tgBotid), msg)
	//msg.ReplyToMessageID = int(tgBotid)
	bot.Send(info)
}

//Here run is a interface method of Job interface
func (j *StatsNotifyJob) Run() {
	if !j.xrayService.IsXrayRunning() {
		return
	}
	var info string
	//get hostname
	name, err := os.Hostname()
	if err != nil {
		fmt.Println("get hostname error:", err)
		return
	}
	info = fmt.Sprintf("Hostname:%s\r\n", name)
	//get ip address
	var ip string
	netInterfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("net.Interfaces failed, err:", err.Error())
		return
	}

	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()

			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						ip = ipnet.IP.String()
						break
					} else {
						ip = ipnet.IP.String()
						break
					}
				}
			}
		}
	}
	info += fmt.Sprintf("IP:%s\r\n \r\n", ip)

	//get traffic
	inbouds, err := j.inboundService.GetAllInbounds()
	if err != nil {
		logger.Warning("StatsNotifyJob run failed:", err)
		return
	}
	//NOTE:If there no any sessions here,need to notify here
	//TODO:分节点推送,自动转化格式
	for _, inbound := range inbouds {
		info += fmt.Sprintf("Node name:%s\r\nPort:%d\r\nUpload↑:%s\r\nDownload↓:%s\r\nTotal:%s\r\n", inbound.Remark, inbound.Port, common.FormatTraffic(inbound.Up), common.FormatTraffic(inbound.Down), common.FormatTraffic((inbound.Up + inbound.Down)))
		if inbound.ExpiryTime == 0 {
			info += fmt.Sprintf("Expire date:unlimited\r\n \r\n")
		} else {
			info += fmt.Sprintf("Expire date:%s\r\n \r\n", time.Unix((inbound.ExpiryTime/1000), 0).Format("2006-01-02 15:04:05"))
		}
	}
	j.SendMsgToTgbot(info)
}

func (j *StatsNotifyJob) UserLoginNotify(username string, ip string, time string, status LoginStatus) {
	if username == "" || ip == "" || time == "" {
		logger.Warning("UserLoginNotify failed,invalid info")
		return
	}
	var msg string
	//get hostname
	name, err := os.Hostname()
	if err != nil {
		fmt.Println("get hostname error:", err)
		return
	}
	if status == LoginSuccess {
		msg = fmt.Sprintf("Successfully logged-in to the panel\r\nHostname:%s\r\n", name)
	} else if status == LoginFail {
		msg = fmt.Sprintf("Login to the panel was unsuccessful\r\nHostname:%s\r\n", name)
	}
	msg += fmt.Sprintf("Time:%s\r\n", time)
	msg += fmt.Sprintf("Username:%s\r\n", username)
	msg += fmt.Sprintf("IP:%s\r\n", ip)
	j.SendMsgToTgbot(msg)
}


var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
    tgbotapi.NewInlineKeyboardRow(
        tgbotapi.NewInlineKeyboardButtonData("✳️ دریافت اطلاعات اکانت", "get_usage"),
    ),
)

func (j *StatsNotifyJob) OnReceive() *StatsNotifyJob {
	tgBottoken, err := j.settingService.GetTgBotToken()
	if err != nil || tgBottoken == "" {
		logger.Warning("sendMsgToTgbot failed,GetTgBotToken fail:", err)
		return j
	}
	bot, err := tgbotapi.NewBotAPI(tgBottoken)
	if err != nil {
		fmt.Println("get tgbot error:", err)
		return j
	}
	bot.Debug = false
	u := tgbotapi.NewUpdate(0)
    u.Timeout = 10

    updates := bot.GetUpdatesChan(u)

    for update := range updates {
        if update.Message == nil { 
			
			if update.CallbackQuery != nil {
				// Respond to the callback query, telling Telegram to show the user
				// a message with the data received.
				callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
				if _, err := bot.Request(callback); err != nil {
					logger.Warning(err)
				}
	
				// And finally, send a message containing the data received.
				msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "")

				switch update.CallbackQuery.Data {
					case "get_usage":
						msg.Text = "برای دریافت اطلاعات سرویس آیدی V2RAY  خود را وارد نمایید: \n <code>/usage uuid</code> \n بطور مثال : <code>/usage fc3239ed-8151-ff51-b183d5182142</code>"
						msg.ParseMode = "HTML"
					}
				if _, err := bot.Send(msg); err != nil {
					logger.Warning(err)
				}
			}
		
            continue
        }

        if !update.Message.IsCommand() { // ignore any non-command Messages
            continue
        }

        // Create a new MessageConfig. We don't have text yet,
        // so we leave it empty.
        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

        // Extract the command from the Message.
        switch update.Message.Command() {
        case "/Trafiic":
            msg.Text = "♻️به بخش مشاهده اطلاعات اشتراک خوش آمدید \n از دکمه زیر برای دریافت اطلاعات اکانت خود استفاده نمایید"
			msg.ReplyMarkup = numericKeyboard

        case "usage":
            msg.Text = j.getClientUsage(update.Message.CommandArguments())
        default:
            msg.Text = "I don't know that command, /help"
			msg.ReplyMarkup = numericKeyboard

        }

        if _, err := bot.Send(msg); err != nil {
            logger.Warning(err)
        }
    }
	return j

}
func (j *StatsNotifyJob) getClientUsage(id string) string {
	traffic , err := j.inboundService.GetClientTrafficById(id)
	if err != nil {
		logger.Warning(err)
		return "⚠️ در ورود اطلاعات اشتباهی رخ داده است"
	}
	expiryTime := ""
	if traffic.ExpiryTime == 0 {
		expiryTime = fmt.Sprintf("unlimited")
	} else {
		expiryTime = fmt.Sprintf("%s", time.Unix((traffic.ExpiryTime/1000), 0).Format("2006-01-02 15:04:05"))
	}
	total := ""
	if traffic.Total == 0 {
		total = fmt.Sprintf("unlimited")
	} else {
		total = fmt.Sprintf("%s", common.FormatTraffic((traffic.Total)))
	}
	output := fmt.Sprintf("♻️ وضعیت فیلترشکن شما : %t\r\n📧 ایمیل ثبت شده : %s\r\n🔼 میزان آپلود ↑ : %s\r\n🔽 میزان دانلود ↓: %s\r\n🔄 حجم کلی : %s / %s\r\n📅 تاریخ اتمام : %",
	traffic.Enable, traffic.Email, common.FormatTraffic(traffic.Up), common.FormatTraffic(traffic.Down), common.FormatTraffic((traffic.Up + traffic.Down)),
	total, expiryTime)
	
	return output
}
