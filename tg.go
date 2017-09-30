package main

import (
    "flag"
    "fmt"
    "encoding/xml"
    "encoding/json"
    "io"
    "io/ioutil"
    _ "log"
    "math/rand"
    "net/http"
    "net/url"
    "os"
    "regexp"
    "strconv"
    "strings"
    "time"
    "github.com/antonholmquist/jason"
    "github.com/beevik/etree"
    "github.com/jbrodriguez/mlog"
    "github.com/Jeffail/gabs"
    "gopkg.in/telegram-bot-api.v4"
    "golang.org/x/exp/utf8string"
    "github.com/coreos/pkg/flagutil"
    "github.com/dghubble/go-twitter/twitter"
    "github.com/dghubble/oauth1"
    "github.com/PuerkitoBio/goquery"
)

var (
    version   string
    builddate string
)

type Result struct {
    WeatherReport []WeatherReport
}

type WeatherReport struct {
    Area string
    TemperatureInformation TemperatureInformation
    RelativeHumidityInformation RelativeHumidityInformation
    WindInformation WindInformation
}

type TemperatureInformation struct {
    Type string
    Measure string
}

type RelativeHumidityInformation struct {
    Type string
    Measure string
}

type WindInformation struct {
    DirectionCode string
    Measure string
}

func random(min, max int) int {
    rand.Seed(time.Now().Unix())
    return rand.Intn(max - min) + min
}

func Download(id string) {
    url := "http://samsungmobile.accu-weather.com/widget/samsungmobile/weather-data.asp?location=cityid:" + id + "&metric=1&langid=12"
    response, e := http.Get(url)
    if e != nil {
        mlog.Fatal(e)
    }

    defer response.Body.Close()

    file, err := os.Create("tmp/" + id + ".xml")
    if err != nil {
        mlog.Fatal(err)
    }
    _, err = io.Copy(file, response.Body)
    if err != nil {
        mlog.Fatal(err)
    }
    file.Close()
}

func Parse(id string) (string, string, string, string) {
    doc := etree.NewDocument()
    if err := doc.ReadFromFile("tmp/" + id + ".xml"); err != nil {
        mlog.Fatal(err)
    }
    
    root := doc.SelectElement("adc_database")
    
    var temperature, realfeel, humidity, weathertext string
    
    for _, weather := range root.SelectElements("currentconditions") {
        if temperature = weather.SelectElement("temperature").Text(); temperature != "" {
            realfeel = weather.SelectElement("realfeel").Text()
            humidity = weather.SelectElement("humidity").Text()
            weathertext = weather.SelectElement("weathertext").Text()
        }
    }
    return temperature, realfeel, humidity, weathertext
}

func HKODownload() {
    //Region XML
    url := "http://www.hko.gov.hk/wxinfo/json/region2.xml"

    response, e := http.Get(url)
    if e != nil {
        mlog.Fatal(e)
    }

    defer response.Body.Close()

    file, err := os.Create("tmp/region2.xml")
    if err != nil {
        mlog.Fatal(err)
    }

    _, err = io.Copy(file, response.Body)
    if err != nil {
        mlog.Fatal(err)
    }
    file.Close()
    
    //Weather Text
    url2 := "http://www.hko.gov.hk/wxinfo/json/fcartoon_json.xml"

    response2, e := http.Get(url2)
    if e != nil {
        mlog.Fatal(e)
    }

    defer response2.Body.Close()

    file2, err := os.Create("tmp/fcartoon_json.xml")
    if err != nil {
        mlog.Fatal(err)
    }

    _, err = io.Copy(file2, response2.Body)
    if err != nil {
        mlog.Fatal(err)
    }
    file2.Close()
}

func HKOParse(str string, regional []byte) (string) {
    var name string
    var area string
    var weathertext string
    var msg string

    v, _ := jason.NewObjectFromBytes(regional)
                
    others, _ := v.GetObject("地區")
        
    for index, value := range others.Map() {
        s, sErr := value.String()

        if sErr == nil {
            if strings.Contains(str, index) {
                area = index
                name = s
            }
        }
    }

    content2, err := ioutil.ReadFile("tmp/fcartoon_json.xml")
    if err != nil {
        mlog.Fatal(err)
    }
    v3, _ := jason.NewObjectFromBytes(content2)
    
    weathericon, _ := v3.GetString("FCARTOON", "Icon1")
    
    content3, err := ioutil.ReadFile("icon.json")
    if err != nil {
        mlog.Fatal(err)
    }
    v2, _ := jason.NewObjectFromBytes(content3)
    
    weathertext, _ = v2.GetString(weathericon)
    
    content4, err := ioutil.ReadFile("tmp/region2.xml")
    if err != nil {
        mlog.Fatal(err)
    }
    var result Result
    err = xml.Unmarshal(content4, &result)
    if err != nil {
        mlog.Fatal(err)
    }
    
    for i := 0;  i<=36; i++ {
        if strings.Contains(name, result.WeatherReport[i].Area) {
            if result.WeatherReport[i].RelativeHumidityInformation.Measure == "" {
                msg = "🌞現時天氣： " + weathertext + "\n🌡現時溫度： " + result.WeatherReport[i].TemperatureInformation.Measure + "°C\n(上面係" + area + "嘅溫度)\n☔️相對濕度： 未能提供\n\n天氣資料係由 F9 提供🐕"
            } else {
                msg = "🌞現時天氣： " + weathertext + "\n🌡現時溫度： " + result.WeatherReport[i].TemperatureInformation.Measure + "°C\n(上面係" + area + "嘅溫度)\n☔️相對濕度： " + result.WeatherReport[i].RelativeHumidityInformation.Measure + "%\n\n天氣資料係由 F9 提供🐕"
            }
        }
    }
    
    return msg
}

func warning() (string) {
    url := "http://www.weather.gov.hk/textonly/v2/warning/warnc.htm"

    response, err := http.Get(url)
    if err != nil {
        mlog.Fatal(err)
    }

    defer response.Body.Close()

    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        mlog.Fatal(err)
    }

    var text string

    if strings.Contains(string(body), "<!--生 效 警 告-->") {
        text = strings.Replace(strings.Replace(strings.Split(strings.Split(string(body), "<!--生 效 警 告-->")[1], "<!--/生 效 警 告-->")[0], "<p>", "<b>", -1), "</p>", "</b>", -1)
    } else {
        text = "現 時 並 無 警 告 生 效。"
    }

    return text
}

func typhoonText() (string) {
    var text string

    doc, err := goquery.NewDocument("http://www.weather.gov.hk/wxinfo/currwx/tc_fixarea_c.htm")
    if err != nil {
        mlog.Fatal(err)
    }

    name := doc.Find(".skin_main_table_td02_table_class h1").First().Text()
    time := doc.Find(".skin_main_table_td02_table_class span").First().Text()
    location := doc.Find(".skin_main_table_td02_table_class table").Eq(4).Find("tr").First().Text()
    wspeed := doc.Find(".skin_main_table_td02_table_class table").Eq(4).Find("tr").Eq(1).Text()
    move := doc.Find(".skin_main_table_td02_table_class table").Eq(4).Find("tr").Eq(2).Text()

    if name == "熱帶氣旋位置及路徑圖" {
        return "現 時 並 無 熱 帶 氣 旋。"
    }

    text = name + "\n" + time + "\n" + strings.Replace(location, "\n", "", -1) + "\n" + strings.Replace(wspeed, "\n", "", -1) + "\n" + strings.Replace(move, "\n", "", -1)

    return text
}

func typhoonImg() (string) {
    url := "http://www.weather.gov.hk/wxinfo/currwx/tc_fixarea_c.htm"

    response, e := http.Get(url)
    if e != nil {
        mlog.Fatal(e)
    }

    defer response.Body.Close()

    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        mlog.Fatal(e)
    }

    var imgurl string

    if strings.Contains(string(body), "熱 帶 氣 旋 路 徑") {
        imgurl = "http://www.weather.gov.hk/wxinfo/currwx/" + strings.Split(strings.Split(string(body), "<p><img src='")[1], "' alt='熱 帶 氣 旋 路 徑'>")[0]

        resp, ee := http.Get(imgurl)
        if ee != nil {
            mlog.Fatal(ee)
        }

        defer resp.Body.Close()

        file, err := os.Create("tmp/typhoon.png")
        if err != nil {
            mlog.Fatal(err)
        }

        _, err = io.Copy(file, resp.Body)
        if err != nil {
            mlog.Fatal(err)
        }
        file.Close()
    }

    return imgurl
}

func RadarDownloader() (string) {
    url := "http://www.hko.gov.hk/wxinfo/radars/radar64n_uc.htm?&"

    response, e := http.Get(url)
    if e != nil {
        mlog.Fatal(e)
    }

    defer response.Body.Close()

    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return "not found"
    }
    imgurl := "http://www.hko.gov.hk/wxinfo/radars/" + strings.Split(strings.Split(string(body), "picture[2][19]=\"")[1], "\";")[0]

    resp, ee := http.Get(imgurl)
    if ee != nil {
        mlog.Fatal(ee)
    }

    defer resp.Body.Close()

    file, err := os.Create("tmp/radar.jpg")
    if err != nil {
        mlog.Fatal(err)
    }

    _, err = io.Copy(file, resp.Body)
    if err != nil {
        mlog.Fatal(err)
    }
    file.Close()
    return imgurl
}

func RadarDownloader256() (string) {
    url := "http://www.hko.gov.hk/wxinfo/radars/radar256n_uc.htm?&"

    response, e := http.Get(url)
    if e != nil {
        mlog.Fatal(e)
    }

    defer response.Body.Close()

    body, err := ioutil.ReadAll(response.Body)
    if err != nil {
        return "not found"
    }
    imgurl := "http://www.hko.gov.hk/wxinfo/radars/" + strings.Split(strings.Split(string(body), "picture[0][9]=\"")[1], "\";")[0]

    resp, ee := http.Get(imgurl)
    if ee != nil {
        mlog.Fatal(ee)
    }

    defer resp.Body.Close()

    file, err := os.Create("tmp/radar256.jpg")
    if err != nil {
        mlog.Fatal(err)
    }

    _, err = io.Copy(file, resp.Body)
    if err != nil {
        mlog.Fatal(err)
    }
    file.Close()
    return imgurl
}

func CheckCFurl(str string, site []byte) (bool) {
    var status bool

    v, _ := jason.NewObjectFromBytes(site)
    
    contentfarm, _ := v.GetStringArray("site")
    for _, cfsite := range contentfarm {
        if strings.Contains(str, cfsite) {
            fmt.Printf("Content Farm Detector: %s\n", cfsite)
            status = true
            continue
        }
    }
    return status
}

func GetRunTime(a, b time.Time) (year, month, day, hour, min, sec int) {
    if a.Location() != b.Location() {
        b = b.In(a.Location())
    }
    if a.After(b) {
        a, b = b, a
    }
    y1, M1, d1 := a.Date()
    y2, M2, d2 := b.Date()

    h1, m1, s1 := a.Clock()
    h2, m2, s2 := b.Clock()

    year = int(y2 - y1)
    month = int(M2 - M1)
    day = int(d2 - d1)
    hour = int(h2 - h1)
    min = int(m2 - m1)
    sec = int(s2 - s1)

    // Normalize negative values
    if sec < 0 {
        sec += 60
        min--
    }
    if min < 0 {
        min += 60
        hour--
    }
    if hour < 0 {
        hour += 24
        day--
    }
    if day < 0 {
        // days in month:
        t := time.Date(y1, M1, 32, 0, 0, 0, 0, time.UTC)
        day += 32 - t.Day()
        month--
    }
    if month < 0 {
        month += 12
        year--
    }

    return
}

func checkAdmin(Bot *tgbotapi.BotAPI, chat *tgbotapi.Chat, user int) bool {
    var chatconfig = chat.ChatConfig()
    var chatconfigwithuser tgbotapi.ChatConfigWithUser

    chatconfigwithuser.ChatID = chatconfig.ChatID
    chatconfigwithuser.SuperGroupUsername = chatconfig.SuperGroupUsername
    chatconfigwithuser.UserID = user

    member, err := Bot.GetChatMember(chatconfigwithuser)
    if err != nil {
        return false
    } else if member.IsAdministrator() || member.IsCreator() {
        return true
    }
    return false
}

func checkUsername(Bot *tgbotapi.BotAPI, chat *tgbotapi.Chat, user int) string {
    var chatconfig = chat.ChatConfig()
    var chatconfigwithuser tgbotapi.ChatConfigWithUser

    chatconfigwithuser.ChatID = chatconfig.ChatID
    chatconfigwithuser.SuperGroupUsername = chatconfig.SuperGroupUsername
    chatconfigwithuser.UserID = user

    member, err := Bot.GetChatMember(chatconfigwithuser)
    if err != nil {
        return ""
    }
    return member.User.UserName
}

func trafficCheck(previd string) (string, string) {
    url := "https://hketraffic.herokuapp.com/api/v1/incidents?lang=hk"

    response, e := http.Get(url)
    if e != nil {
        mlog.Fatal(e)
    }

    defer response.Body.Close()
    body, _ := ioutil.ReadAll(response.Body)

    if string(body) == "[]" {
        return "運輸署網站未有最新交通消息可供查閱。", "0"
    }

    var dataMap []map[string]interface{}
    json.Unmarshal(body, &dataMap)

    if previd == dataMap[0]["_id"].(string) {
        return "UpToDate", dataMap[0]["_id"].(string)
    }

    newsurl := "https://hketraffic.herokuapp.com/api/v1/incidents/" + dataMap[0]["_id"].(string) + "?lang=hk"

    response2, e := http.Get(newsurl)
    if e != nil {
        mlog.Fatal(e)
    }

    defer response2.Body.Close()
    body2, _ := ioutil.ReadAll(response2.Body)

    v, _ := jason.NewObjectFromBytes(body2)
    title, _ := v.GetString("headline")
    content, _ := v.GetString("content")
    date, _ := v.GetString("publishedDate")

    return title + "\n\n" + content + "\n\n發佈日期: " + date, dataMap[0]["_id"].(string)
}

func MTRUpdate() (string) {
    flags := flag.NewFlagSet("user-auth", flag.ExitOnError)
    consumerKey := flags.String("consumer-key", "", "Twitter Consumer Key")
    consumerSecret := flags.String("consumer-secret", "", "Twitter Consumer Secret")
    accessToken := flags.String("access-token", "", "Twitter Access Token")
    accessSecret := flags.String("access-secret", "", "Twitter Access Secret")
    flags.Parse(os.Args[1:])
    flagutil.SetFlagsFromEnv(flags, "TWITTER")

    if *consumerKey == "" || *consumerSecret == "" || *accessToken == "" || *accessSecret == "" {
        mlog.Fatal("Consumer key/secret and Access token/secret required")
    }

    config := oauth1.NewConfig(*consumerKey, *consumerSecret)
    token := oauth1.NewToken(*accessToken, *accessSecret)
    httpClient := config.Client(oauth1.NoContext, token)

    client := twitter.NewClient(httpClient)

    userTimelineParams := &twitter.UserTimelineParams{ScreenName: "mtrupdate", Count: 10}
    tweets, _, _ := client.Timelines.UserTimeline(userTimelineParams)

    var isBroken string
    for i := 0; i < 10; i++ {
        match, _ := regexp.MatchString(`(訊號故障|維持正常|嚴重受阻|稍有阻延|回復正常|停止服務|紓緩擠迫|紓緩擠塞)`, tweets[i].Text)
        if match {
            isBroken = tweets[i].Text
            break
        } else {
            isBroken = "問問問，你好想壞車咩？"
        }
    }
    return isBroken
}

func secondsToMinutes(Seconds int) string {
    seconds := Seconds % 60
    minutes := Seconds / 60
    hours := minutes / 60
    minutes = minutes % 60
    str := fmt.Sprintf("%d:%d:%d", hours, minutes, seconds)
    return str
}

func main() {
    current_time := time.Now().Local()
    save := current_time.Format("2006-01-02")
    mlog.Start(mlog.LevelInfo, "log/"+save+".log")

    var bot *tgbotapi.BotAPI

	bot, err := tgbotapi.NewBotAPI("yourtokenhere")

    if err != nil {
        mlog.Error(err)
    }

    bot.Debug = false

    mlog.Info("Authorized on account %s", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    var f9talk bool
    var radarID, radarID2, prevURL, prevURL2, nextURL, nextURL2 string
    var radartask, radartask2 bool
    var previd string
    var prevnews string

    //Content Farm Preload
    farmlist, _ := ioutil.ReadFile("cfsite.json")

    //Regional List Preload
    regionallist, _ := ioutil.ReadFile("weather.json")
    regionallist2, _ := ioutil.ReadFile("xml.json")

    //Anti Spam System
    spamConf := gabs.New()
    
    updates, err := bot.GetUpdatesChan(u)

    go func() {
        for {
            unban_time := time.Now().Unix()
            list, _ := jason.NewObjectFromBytes([]byte(spamConf.String()))
            user, err := list.GetObject("user")
            if err == nil {
                for userid, unixtimestamp := range user.Map() {
                    time64, _ := unixtimestamp.Int64()
                    if time64 == unban_time {
                        spamConf.Set(0, "user", userid)
                    }
                }
            }
            time.Sleep(time.Second * 1)
        }
    }()

    go func() {
        for {
            list, _ := jason.NewObjectFromBytes([]byte(spamConf.String()))
            user, err := list.GetObject("user")
            if err == nil {
                for userid, count := range user.Map() {
                    time64, _ := count.Int64()
                    if time64 < 10 {
                        spamConf.Set(0, "user", userid)
                    }
                }
            }
            time.Sleep(time.Second * 30)
        }
    }()

    go func() {
        prevURL = RadarDownloader()
        msg := tgbotapi.NewPhotoUpload(-1001142945893, "tmp/radar.jpg")
        log, _ := bot.Send(msg)
        radarID = (*log.Photo)[0].FileID
        for {
            current_time := time.Now().Local()
            if current_time.Minute() == 2 || current_time.Minute() == 8 || current_time.Minute() == 14 || current_time.Minute() == 20 || current_time.Minute() == 26 || current_time.Minute() == 32 || current_time.Minute() == 38 || current_time.Minute() == 44 || current_time.Minute() == 50 || current_time.Minute() == 56 || radartask {
                nextURL = RadarDownloader()
                if prevURL == nextURL {
                    radartask = true
                    time.Sleep(time.Second * 15)
                    continue
                }
                msg := tgbotapi.NewPhotoUpload(-1001142945893, "tmp/radar.jpg")
                log, _ := bot.Send(msg)
                radarID = (*log.Photo)[0].FileID
                radartask = false
            }
            time.Sleep(time.Second * 60)
        }
    }()

    go func() {
        prevURL2 = RadarDownloader256()
        msg := tgbotapi.NewPhotoUpload(-1001106072975, "tmp/radar256.jpg")
        log, _ := bot.Send(msg)
        radarID2 = (*log.Photo)[0].FileID
        for {
            current_time := time.Now().Local()
            if current_time.Minute() == 14 || current_time.Minute() == 26 || current_time.Minute() == 38 || current_time.Minute() == 50 || current_time.Minute() == 2 || radartask2 {
                nextURL2 = RadarDownloader256()
                if prevURL2 == nextURL2 {
                    radartask2 = true
                    time.Sleep(time.Second * 15)
                    continue
                }
                msg := tgbotapi.NewPhotoUpload(-1001106072975, "tmp/radar256.jpg")
                log, _ := bot.Send(msg)
                radarID2 = (*log.Photo)[0].FileID
                radartask2 = false
            }
            time.Sleep(time.Second * 60)
        }
    }()

    for update := range updates {
        if update.EditedMessage != nil {

            if update.EditedMessage.IsCommand() {
                //Binc Process
                bincrate := random(1, 100)
                if update.EditedMessage.Command() == "binc" {
                    if bincrate >= 20 && bincrate <= 30 || update.EditedMessage.From.ID == 89714653 {
                        msg := tgbotapi.NewMessage(update.EditedMessage.Chat.ID, "#追數list:\n@demkeela pixel\n@phidias0303 4K mon, Marco Polo Club 黑卡, M12牛\nAgentCC 田田洗完黑錢買比我地 iphone 7/7 plus\n@diuleilomooooooooooooooooooooooo 自拍同台妹扑嘢.jpg/.png/.mkv/.mp4/.3gp/other media file\n\n#鞭屍list: #集中營二寶\n@Kenchan95  冇linux邊有mac🙄\n@snoopy1344 我睇緊架 你地唔洗叫我\n待補 歡迎報名\n\n#F9專用List:\n@husky7x24  死都唔買R* game\nEA games冇bug 我一次過買晒ea d game 是但一隻驚刀duck")
                        bot.Send(msg)
                    }
                    continue
                }
                
                //2017 Dead List
                deadrate := random(1, 100)
                if update.EditedMessage.Command() == "deadlist" {
                    if deadrate >= 40 && deadrate <= 60 || update.EditedMessage.From.ID == 89714653 {
                        msg := tgbotapi.NewMessage(update.EditedMessage.Chat.ID, "#2017DeadList #RIP\nnyaa\napricity\nhackpad\nnacx\nebbio\nmyavsuper\n18deny")
                        bot.Send(msg)
                    }
                    continue
                }

                //Useless
                if update.EditedMessage.Command() == "islovear" && update.EditedMessage.CommandArguments() != "" {
                    msg := tgbotapi.NewMessage(update.EditedMessage.Chat.ID, "係愛呀 " + update.EditedMessage.CommandArguments())
                    bot.Send(msg)
                    continue
                }
            }
            
            //CKbb
            if strings.Contains(update.EditedMessage.Text, "CKbb") {
                chance := random(1, 100)
                if chance >= 1 && chance <= 10 {
                    msg := tgbotapi.NewVoiceShare(update.EditedMessage.Chat.ID, "AwADBQADLAADJ1wJVrzRf96cVPcnAg")
                    msg.ReplyToMessageID = update.EditedMessage.MessageID
                    bot.Send(msg)
                }
                continue
            }
            
            //Tag Bu5hit
            if strings.Contains(update.EditedMessage.Text, "GCF9") {
                gcrate := random(1, 100)
                if gcrate >= 1 && gcrate <= 50 || update.EditedMessage.From.ID == 89714653 || update.EditedMessage.From.ID == 11457427 {
                    msg := tgbotapi.NewMessage(update.EditedMessage.Chat.ID, "@bu5hit")
                    msg.ReplyToMessageID = update.EditedMessage.MessageID
                    bot.Send(msg)
                }
                continue
            }
            
            //Content Farm Process
            match, _ := regexp.MatchString(`(http[s]?)://([\w\-_]+(\.[\w\-_]+){0,5}(:\d+)?)\.[a-zA-Z]{2,12}`, update.EditedMessage.Text)
            if match {
                var IsFarm bool
                var isAdmin bool

                IsFarm = CheckCFurl(update.EditedMessage.Text, farmlist)
                isAdmin = checkAdmin(bot, update.EditedMessage.Chat, bot.Self.ID)

                if IsFarm {
                    alert := "請不要分享震驚十三億人的內容農場，謝謝🐕"
                    if !isAdmin {
                        alert = alert + "\n由於Husky Bot不是群組管理員，無法刪除相關訊息。"
                    }
                    warn := tgbotapi.NewMessage(update.EditedMessage.Chat.ID, alert)
                    warn.ReplyToMessageID = update.EditedMessage.MessageID
                    bot.Send(warn)
                    continue
                }
                
                if strings.Contains(update.EditedMessage.Text, "unwire.hk") {
                    msg2 := tgbotapi.NewMessage(update.EditedMessage.Chat.ID, "睇少D underwear啦屌你🐕")
                    msg2.ReplyToMessageID = update.EditedMessage.MessageID
                    bot.Send(msg2)
                    continue
                }
            }
            
            if strings.Contains(update.EditedMessage.Text, "F9") || strings.Contains(update.EditedMessage.Text, "f9") || strings.Contains(update.EditedMessage.Caption, "f9") || strings.Contains(update.EditedMessage.Caption, "F9") {
                if update.EditedMessage.From.ID == 89714653 {
                    continue
                }
                t := time.Now()
    
                if t.Hour() < 7 {
                    getrate := random(1, 100)
                    if getrate >= 10 && getrate <= 25 {
                        msg := tgbotapi.NewMessage(update.EditedMessage.Chat.ID, "訓啦柒頭\n依家時間係" + t.Format("3:04PM") + "\nF9訓緊教\n咪嘈啦屌\n阻撚住晒😴")
                        msg.ReplyToMessageID = update.EditedMessage.MessageID
                        bot.Send(msg)
                    }
                    continue
                }
                chance := random(1, 100)
                if chance >= 1 && chance <= 20 {
                    msg := tgbotapi.NewMessage(update.EditedMessage.Chat.ID, "dllm\neditllm\n當F9冇到？")
                    msg.ReplyToMessageID = update.EditedMessage.MessageID
                    bot.Send(msg)
                } else if chance >= 21 && chance <= 40 {
                    msg := tgbotapi.NewMessage(update.EditedMessage.Chat.ID, "關你撚事\neditllm\n當F9冇到？")
                    msg.ReplyToMessageID = update.EditedMessage.MessageID
                    bot.Send(msg)
                } else if chance >= 41 && chance <= 60 {
                    msg := tgbotapi.NewStickerShare(update.EditedMessage.Chat.ID, "CAADBQADYwUAAmQK4AW3jYFjDvykkAI")
                    msg.ReplyToMessageID = update.EditedMessage.MessageID
                    bot.Send(msg)
                } else if chance >= 61 && chance <= 80 {
                    msg := tgbotapi.NewStickerShare(update.EditedMessage.Chat.ID, "CgADBAAD6UIAAiceZAeFb3k7aPuhyQI")
                    msg.ReplyToMessageID = update.EditedMessage.MessageID
                    bot.Send(msg)
                } else {
                    msg := tgbotapi.NewMessage(update.EditedMessage.Chat.ID, "#追數list #鞭屍list #F9專用List\neditllm\n當F9冇到？")
                    msg.ReplyToMessageID = update.EditedMessage.MessageID
                    bot.Send(msg)
                }
            }
        } else if update.Message != nil {
            //Command Process
            if update.Message.IsCommand() {
                //Get Bot Version
                if update.Message.Command() == "info" {
                    _, _, day, hour, min, sec := GetRunTime(current_time, time.Now().Local())
                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Husky Bot v" + version + "\nLanguage: Go\nLibrary: [go-telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api)\nUptime: " + strconv.Itoa(day) + " Days " + strconv.Itoa(hour) + " Hours " + strconv.Itoa(min) + " Minutes " + strconv.Itoa(sec) + " Seconds\nLast Modified Date: " + builddate)
                    msg.ReplyToMessageID = update.Message.MessageID
                    msg.ParseMode = "Markdown"
                    msg.DisableWebPagePreview = true
                    bot.Send(msg)
                    continue
                }

                //Binc Process
                bincrate := random(1, 100)
                if update.Message.Command() == "binc" {
                    if bincrate >= 20 && bincrate <= 30 || update.Message.From.ID == 89714653 {
                        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "#追數list:\n@decaf_asm pixel\n@phidias0303 4K mon, Marco Polo Club 黑卡, M12牛\nAgentCC 田田洗完黑錢買比我地 iphone 7/7 plus\n@Hacker18deny 自拍同台妹扑嘢.jpg/.png/.mkv/.mp4/.3gp/other media file\n\n#鞭屍list: #集中營二寶\n@Kenchan95  冇linux邊有mac🙄\n@snoopy1344 我睇緊架 你地唔洗叫我\n待補 歡迎報名\n\n#F9專用List:\n@husky7x24  死都唔買R* game\nEA games冇bug 我一次過買晒ea d game 是但一隻驚刀duck")
                        bot.Send(msg)
                    }
                    continue
                }

                //Useless
                if update.Message.Command() == "islovear" && update.Message.CommandArguments() != "" {
                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "係愛呀 " + update.Message.CommandArguments())
                    bot.Send(msg)
                    continue
                }

                //Get Command
                if update.Message.Command() == "admin" {
                    UserID, err := strconv.Atoi(update.Message.CommandArguments())
                    if err != nil {
                        continue
                    }

                    var isAdmin bool
                    isAdmin = checkAdmin(bot, update.Message.Chat, UserID)

                    if isAdmin {
                        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "This user(ID: " + update.Message.CommandArguments() + ") is an administrator.")
                        msg.ReplyToMessageID = update.Message.MessageID
                        bot.Send(msg)
                    } else {
                        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "This user(ID: " + update.Message.CommandArguments() + ") isn't an administrator.")
                        msg.ReplyToMessageID = update.Message.MessageID
                        bot.Send(msg)
                    }
                    continue
                }

                if update.Message.Command() == "check" {
                    UserID, err := strconv.Atoi(update.Message.CommandArguments())
                    if err != nil {
                        continue
                    }

                    var username string
                    username = checkUsername(bot, update.Message.Chat, UserID)

                    if username != "" {
                        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "This user(ID: " + update.Message.CommandArguments() + ") username: " + username)
                        msg.ReplyToMessageID = update.Message.MessageID
                        bot.Send(msg)
                    } else {
                        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "This user(ID: " + update.Message.CommandArguments() + ") haven't set username.")
                        msg.ReplyToMessageID = update.Message.MessageID
                        bot.Send(msg)
                    }
                    continue
                }

                if update.Message.Command() == "add" {
                    if update.Message.From.ID == 89714653 && update.Message.CommandArguments() != "" {
                        var exist bool
                        v, _ := gabs.ParseJSON(farmlist)
                        children, _ := v.S("site").Children()
                        for _, child := range children {
                            if child.Data().(string) == update.Message.CommandArguments() {
                                exist = true
                                msg := tgbotapi.NewMessage(update.Message.Chat.ID, "The Conten Farm Website(" + update.Message.CommandArguments() + ") you are trying to add is already in the list.")
                                msg.ReplyToMessageID = update.Message.MessageID
                                bot.Send(msg)
                                break
                            }
                        }
                        if exist { continue }
                        v.ArrayAppend(update.Message.CommandArguments(), "site")
                        fmt.Println("Content Farm Blocker: " + update.Message.CommandArguments() + " has been added.")
                        _ = ioutil.WriteFile("cfsite.json", []byte(v.StringIndent("", "  ")), 0644)
                        farmlist, _ = ioutil.ReadFile("cfsite.json")
                        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "The Conten Farm Website(" + update.Message.CommandArguments() + ") has been added.")
                        msg.ReplyToMessageID = update.Message.MessageID
                        bot.Send(msg)
                    }
                    continue
                }

                if update.Message.Command() == "feature" {
                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Husky Bot v" + version + "\n主要功能: \n1. Husky系統\n2. 查詢香港地區天氣\n    查詢天氣，請輸入：地區名稱+咩天氣\n    e.g., 香港咩天氣\n    如沒有該地區天氣，請查詢鄰近地區。\n3. 查詢最新 天氣警告(/warning) 或 交通消息(/traffic)\n4. 封鎖內容農場\n    自動刪除包含內容農場網站的訊息\n    *此功能只適用於Husky Bot為群組管理員*\n    如發現其他內容農場網站，請使用 /report 回報。\n    回報格式: /report URL\n    e.g., /report http://example.com\n5. 隱藏功能，請自行發掘")
                    msg.ReplyToMessageID = update.Message.MessageID
                    msg.ParseMode = "Markdown"
                    bot.Send(msg)
                    continue
                }

                if update.Message.Command() == "report" {
                    count, _ := spamConf.Search("user", strconv.Itoa(update.Message.From.ID)).Data().(float64)
                    if len(strconv.FormatFloat(count, 'f', 0, 64)) == 10 {
                        continue
                    } else if count >= 10 {
                        bantime := time.Now().Unix()
                        spamConf.Set(bantime+300, "user", strconv.Itoa(update.Message.From.ID))
                        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "請勿濫用內容農場回報系統。")
                        msg.ReplyToMessageID = update.Message.MessageID
                        bot.Send(msg)
                        continue
                    } else {
                        spamConf.Set(count+1, "user", strconv.Itoa(update.Message.From.ID))
                        match, err := url.Parse(update.Message.CommandArguments())
                        if err != nil {
                            continue
                        }
                        isASCII := utf8string.NewString(match.Host).IsASCII()
                        isValidTLD, _ := regexp.MatchString("\\.[a-zA-Z]{2,12}", match.Host)
                        if match.Scheme == "http" && isASCII && isValidTLD || match.Scheme == "https" && isASCII && isValidTLD {
                            msg := tgbotapi.NewMessage(update.Message.Chat.ID, "感謝您的回報，Husky會盡快處理。")
                            msg.ReplyToMessageID = update.Message.MessageID
                            bot.Send(msg)
                            if strings.Contains(match.String(), "husky") {
                                continue
                            }
                            forward := tgbotapi.NewForward(89714653, update.Message.Chat.ID, update.Message.MessageID)
                            bot.Send(forward)
                        }
                    }
                    continue
                }

                if update.Message.Command() == "remove" {
                    kbd := tgbotapi.ReplyKeyboardMarkup{
                        Selective:       true,
                        OneTimeKeyboard: true,
                        ResizeKeyboard:  true,
                        Keyboard: [][]tgbotapi.KeyboardButton{
                            []tgbotapi.KeyboardButton{
                                {"I Love Husky🐕", false, false},
                            },
                        },
                    }

                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🐕")
                    msg.ReplyToMessageID = update.Message.MessageID
                    msg.ReplyMarkup = kbd
                    bot.Send(msg)

                    msg2 := tgbotapi.NewMessage(update.Message.Chat.ID, "🐕")
                    msg2.ReplyToMessageID = update.Message.MessageID
                    msg2.ReplyMarkup = tgbotapi.NewRemoveKeyboard(false)
                    bot.Send(msg2)
                    continue
                }
            }
            
            //CKbb
            if strings.Contains(update.Message.Text, "CKbb") {
                chance := random(1, 100)
                if chance >= 1 && chance <= 10 {
                    msg := tgbotapi.NewVoiceShare(update.Message.Chat.ID, "AwADBQADLAADJ1wJVrzRf96cVPcnAg")
                    msg.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg)
                }
                continue
            }
            
            //Tag Bu5hit
            if strings.Contains(update.Message.Text, "GCF9") {
                gcrate := random(1, 100)
                if gcrate >= 1 && gcrate <= 50 || update.Message.From.ID == 89714653 || update.Message.From.ID == 11457427 {
                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "@bu5hit")
                    msg.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg)
                }
                continue
            }
            
            //CS1.6
            if strings.Contains(update.Message.Text, "CS1.6") {
                gcrate := random(1, 100)
                if gcrate >= 50 && gcrate <= 100 || update.Message.From.ID == 89714653 {
                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "CS1.6 the best")
                    msg.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg)
                }
                continue
            }
            
            //Content Farm Process
            match, _ := regexp.MatchString(`(http[s]?)://([\w\-_]+(\.[\w\-_]+){0,5}(:\d+)?)\.[a-zA-Z]{2,12}`, update.Message.Text)
            if match {
                var IsFarm bool
                var isAdmin bool

                IsFarm = CheckCFurl(update.Message.Text, farmlist)
                isAdmin = checkAdmin(bot, update.Message.Chat, bot.Self.ID)
                
                if IsFarm {
                    alert := "請不要分享震驚十三億人的內容農場，謝謝🐕"
                    if !isAdmin {
                        alert = alert + "\n由於Husky Bot不是群組管理員，無法刪除相關訊息。"
                    }
                    warn := tgbotapi.NewMessage(update.Message.Chat.ID, alert)
                    warn.ReplyToMessageID = update.Message.MessageID
                    bot.Send(warn)
                    continue
                }
                
                if strings.Contains(update.Message.Text, "unwire.hk") {
                    msg2 := tgbotapi.NewMessage(update.Message.Chat.ID, "睇少D underwear啦屌你🐕")
                    msg2.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg2)
                    continue
                }

                if strings.Contains(update.Message.Text, "weekendhk.com") {
                    msg2 := tgbotapi.NewMessage(update.Message.Chat.ID, "睇少D鳩假期啦屌你🐕")
                    msg2.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg2)
                    continue
                }
            }
            
            //F9 Process
            var sticker bool
            var gif bool
            var voice bool
            if update.Message.Sticker != nil {
                if update.Message.Sticker.FileID == "CAADBQAD4AEAApiFBgnxQecBXOhbBwI" || update.Message.Sticker.FileID == "CAADBQADSQADbszrEBZalZpGSwMoAg"{
                    sticker = true
                } else {
                    sticker = false
                }
            } else {
                sticker = false
            }
            
            if update.Message.Document != nil {
                if update.Message.Document.FileID == "CgADBQADEwADSINqBtq2O1aaz-H9Ag" || update.Message.Document.FileID == "CgADBAAD5zYAAowdZAcyNMHGyIorcAI" {
                    gif = true
                } else {
                    gif = false
                }
            } else {
                gif = false
            }
            
            if update.Message.Voice != nil {
                if update.Message.Voice.FileID == "AwADBQADBAADbr6QVbRCj8fHpV8BAg" {
                    voice = true
                } else {
                    voice = false
                }
            } else {
                voice = false
            }
            
            if strings.Contains(update.Message.Text, "F9") || strings.Contains(update.Message.Text, "f9") || strings.Contains(update.Message.Caption, "f9") || strings.Contains(update.Message.Caption, "F9") || sticker || gif || voice {
                if update.Message.From.ID == 89714653 {
                    continue
                }
                t := time.Now()
    
                if t.Hour() < 7 {
                    getrate := random(1, 100)
                    if getrate >= 10 && getrate <= 25 {
                        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "訓啦柒頭\n依家時間係" + t.Format("3:04PM") + "\nF9訓緊教\n咪嘈啦屌\n阻撚住晒😴")
                        msg.ReplyToMessageID = update.Message.MessageID
                        bot.Send(msg)
                    }
                    continue
                }
                
                if t.Hour() >= 7 && f9talk {
                    shdpush := random(1, 100)
                    if shdpush >= 1 && shdpush <=60 {
                        chance := random(1, 100)
                        if chance >= 1 && chance <= 20 {
                            msg := tgbotapi.NewMessage(update.Message.Chat.ID, "dllm")
                            msg.ReplyToMessageID = update.Message.MessageID
                            bot.Send(msg)
                        } else if chance >= 21 && chance <= 40 {
                            msg := tgbotapi.NewMessage(update.Message.Chat.ID, "關你撚事")
                            msg.ReplyToMessageID = update.Message.MessageID
                            bot.Send(msg)
                        } else if chance >= 41 && chance <= 60 {
                            msg := tgbotapi.NewStickerShare(update.Message.Chat.ID, "CAADBQADYwUAAmQK4AW3jYFjDvykkAI")
                            msg.ReplyToMessageID = update.Message.MessageID
                            bot.Send(msg)
                        } else if chance >= 61 && chance <= 80 {
                            msg := tgbotapi.NewStickerShare(update.Message.Chat.ID, "CgADBAAD6UIAAiceZAeFb3k7aPuhyQI")
                            msg.ReplyToMessageID = update.Message.MessageID
                            bot.Send(msg)
                        } else {
                            msg := tgbotapi.NewMessage(update.Message.Chat.ID, "#追數list #鞭屍list #F9專用List")
                            msg.ReplyToMessageID = update.Message.MessageID
                            bot.Send(msg)
                        }
                        continue
                    } else {
                        msg := tgbotapi.NewMessage(update.Message.Chat.ID, "唔好嘈住我訓教得唔得?\n等我訓醒先講!")
                        msg.ReplyToMessageID = update.Message.MessageID
                        bot.Send(msg)
                        f9talk = false
                        continue
                    }
                } else {
                    talkagain := random(1, 100)
                    if talkagain <= 30 {
                        f9talk = true
                    }
                }
            }

            //Traffic News
            if update.Message.Text == "交通消息" || update.Message.Command() == "traffic" {
                report, newsid := trafficCheck(previd)
                if report == "UpToDate" {
                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, prevnews)
                    msg.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg)
                } else {
                    previd = newsid
                    prevnews = report
                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, report)
                    msg.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg)
                }
                continue
            }

            //MTR Update
            if update.Message.Text == "地鐵壞車" || update.Message.Command() == "mtr" {
                badnews := MTRUpdate()
                msg := tgbotapi.NewMessage(update.Message.Chat.ID, badnews)
                msg.ReplyToMessageID = update.Message.MessageID
                bot.Send(msg)
                continue
            }

            //Weather Process
            if strings.Contains(update.Message.Text, "地獄") && strings.Contains(update.Message.Text, "天氣") {
                msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🌞現時天氣：最近天氣開始轉涼了！\n🌡現時溫度大概為 1.417×1032°C\n(上述為地獄平均溫度)\n☔️相對濕度 0%\n\n天氣資料係由 F9 提供🐕")
                msg.ReplyToMessageID = update.Message.MessageID
                bot.Send(msg)
                continue
            }
            
            if strings.Contains(update.Message.Text, "天堂") && strings.Contains(update.Message.Text, "天氣") {
                msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🌞現時天氣：最近天氣開始回暖了！\n🌡現時溫度大概為 273°C\n(上述為天堂平均溫度)\n☔️相對濕度 100%\n\n天氣資料係由 HY 提供🐕")
                msg.ReplyToMessageID = update.Message.MessageID
                bot.Send(msg)
                continue
            }

            if update.Message.Text == "雷達圖" || update.Message.Command() == "radar" {
                if radarID == "" {
                    msg := tgbotapi.NewPhotoUpload(update.Message.Chat.ID, "tmp/radar.jpg")
                    msg.ReplyToMessageID = update.Message.MessageID
                    log, _ := bot.Send(msg)
                    radarID = (*log.Photo)[0].FileID
                } else {
                    msg := tgbotapi.NewPhotoShare(update.Message.Chat.ID, radarID)
                    msg.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg)
                }
                continue
            }

            if update.Message.Text == "雷達圖256" || update.Message.Command() == "radar256" {
                if radarID2 == "" {
                    msg := tgbotapi.NewPhotoUpload(update.Message.Chat.ID, "tmp/radar256.jpg")
                    msg.ReplyToMessageID = update.Message.MessageID
                    log, _ := bot.Send(msg)
                    radarID2 = (*log.Photo)[0].FileID
                } else {
                    msg := tgbotapi.NewPhotoShare(update.Message.Chat.ID, radarID2)
                    msg.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg)
                }
                continue
            }

            if update.Message.Text == "天氣警告" || update.Message.Command() == "warning" {
                report := warning()
                msg := tgbotapi.NewMessage(update.Message.Chat.ID, report)
                msg.ReplyToMessageID = update.Message.MessageID
                msg.ParseMode = "HTML"
                bot.Send(msg)
                continue
            }

            if update.Message.Text == "打風" || update.Message.Command() == "typhoon" {
                imgurl := typhoonImg()
                if imgurl == "" {
                    text := typhoonText()
                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
                    msg.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg)
                } else {
                    text := typhoonText()
                    msg := tgbotapi.NewPhotoUpload(update.Message.Chat.ID, "tmp/typhoon.png")
                    msg.Caption = text
                    msg.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg)
                }
                continue
            }

            if strings.Contains(update.Message.Text, "咩天氣") {
                var found, found2 bool
                v, _ := jason.NewObjectFromBytes(regionallist)
                
                others, _ := v.GetObject("地區")
        
                for index, value := range others.Map() {
                    s, sErr := value.String()

                    if sErr == nil {
                        if strings.Contains(update.Message.Text, index) && strings.Contains(update.Message.Text, "天氣") {
                            Download(s)
                            temperature, realfeel, humidity, weathertext := Parse(s)
                            msg := tgbotapi.NewMessage(update.Message.Chat.ID, "🌞現時天氣： " + weathertext + "\n🌡現時溫度： " + temperature + "°C\n⛄️體感溫度： " + realfeel + "°C\n(上面係" + index + "嘅溫度)\n☔️相對濕度 " + humidity + "\n\n天氣資料係由 F9 提供🐕")
                            msg.ReplyToMessageID = update.Message.MessageID
                            bot.Send(msg)
                            found = true
                            found2 = true
                            break
                        }
                    }
                }
                
                if !found {
                    HKODownload()
                    hkomsg := HKOParse(update.Message.Text, regionallist2)
                    if hkomsg != "" {
                        msg := tgbotapi.NewMessage(update.Message.Chat.ID, hkomsg)
                        msg.ReplyToMessageID = update.Message.MessageID
                        bot.Send(msg)
                        found = true
                        found2 = true
                        continue
                    }
                }
                
                if !found2 {
                    msg := tgbotapi.NewMessage(update.Message.Chat.ID, "F9搵唔到您想要嘅地區天氣🐕")
                    msg.ReplyToMessageID = update.Message.MessageID
                    bot.Send(msg)
                }
            }
        } else if update.CallbackQuery != nil {

        } else {

        }
    }
}