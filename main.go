package main

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"time"
)

const (
	RequestURL     = "https://azal.az/book/api/flights/search/by-deeplink"
	TelegramAPIURL = "https://api.telegram.org/bot%s/sendMessage"
	Version        = "0.1.0"
)

var (
	ErrorNoFlightsAvailable = fmt.Errorf("no flights available")
)

var Colors = struct {
	reset   string
	Red     string
	Green   string
	Yellow  string
	Orange  string
	Blue    string
	Magenta string
	Cyan    string
	Gray    string
	White   string
}{
	reset:   "\033[0m",
	Red:     "\033[31m",
	Green:   "\033[32m",
	Yellow:  "\033[33m",
	Orange:  "\033[38;5;208m",
	Blue:    "\033[34m",
	Magenta: "\033[35m",
	Cyan:    "\033[36m",
	Gray:    "\033[37m",
	White:   "\033[97m",
}

func Colored(color string, a ...any) string {
	return color + fmt.Sprint(a...) + Colors.reset
}

type AvialableFlights map[string][]time.Time

type TelegramRequest struct {
	Client *http.Client
	BotKey string
	ChatID string
}

func (telegramRequest *TelegramRequest) sendTelegramMessage(message string) error {
	url := fmt.Sprintf(TelegramAPIURL, telegramRequest.BotKey)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	q := req.URL.Query()
	q.Add("chat_id", telegramRequest.ChatID)
	q.Add("text", message)
	q.Add("parse_mode", "HTML")
	req.URL.RawQuery = q.Encode()

	resp, err := telegramRequest.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("error: telegram send message status code: %d", resp.StatusCode)
	}
	return nil
}

func (telegramRequest *TelegramRequest) sendTelegramFlightNotification(avialableFlights AvialableFlights) error {
	message := "Azal Bot\n\n"
	for day, flights := range avialableFlights {
		message += fmt.Sprintf("%s\n-----------\n", day)
		for _, flight := range flights {
			message += fmt.Sprintf("%s\n", flight.Format("15:04:05"))
		}
		message += "\n"
	}
	message = message[:len(message)-1]
	return telegramRequest.sendTelegramMessage(message)
}

func (telegramRequest *TelegramRequest) sendTelegramStartNotification(botConfig *BotConfig) error {
	return telegramRequest.sendTelegramMessage(
		fmt.Sprintf(
			"Azal Bot started\n\nFrom: %s\nTo: %s\nFirst Date: %s\nLast Date: %s\nRepetition Interval: %s",
			botConfig.From,
			botConfig.To,
			botConfig.FirstDate.Format("2006-01-02T15:04:05"),
			botConfig.LastDate.Format("2006-01-02T15:04:05"),
			botConfig.RepetInterval.String(),
		),
	)
}

type UserInput struct {
	FirstDate      time.Time
	LastDate       time.Time
	From           string
	To             string
	TelegramBotKey string
	TelegramChatID string
	RepetInterval  time.Duration
}

type BotConfig struct {
	FirstDate     time.Time
	LastDate      time.Time
	From          string
	To            string
	days          []string
	RepetInterval time.Duration
}

type ResponseTime struct {
	time.Time
}

func (responseTime *ResponseTime) UnmarshalJSON(b []byte) error {
	s := string(b)
	s = s[1 : len(s)-1]

	t, err := time.Parse("2006-01-02T15:04:05", s)
	if err != nil {
		return err
	}
	responseTime.Time = t
	return nil
}

type SuccessResponse struct {
	Warnings []any `json:"warnings"`
	Search   struct {
		OptionSets []struct {
			Options []struct {
				ID        string `json:"id"`
				Available bool   `json:"available"`
				Route     struct {
					ID            string       `json:"id"`
					DepartureDate ResponseTime `json:"departureDate"`
				} `json:"route"`
			} `json:"options"`
		} `json:"optionSets"`
	} `json:"search"`
}

type ErrorResponse struct {
	Error struct {
		Code string `json:"code"`
		Text string `json:"text"`
	} `json:"error"`
}

type HeaderConfig struct {
	Host           string `req_header:"Host"`
	UserAgent      string `req_header:"User-Agent"`
	Accept         string `req_header:"Accept"`
	AcceptLanguage string `req_header:"Accept-Language"`
	AcceptEncoding string `req_header:"Accept-Encoding"`
	XApplication   string `req_header:"x-application"`
	XLocale        string `req_header:"x-locale"`
	Connection     string `req_header:"Connection"`
	Referer        string `req_header:"Referer"`
	SecFetchDest   string `req_header:"Sec-Fetch-Dest"`
	SecFetchMode   string `req_header:"Sec-Fetch-Mode"`
	SecFetchSite   string `req_header:"Sec-Fetch-Site"`
	TE             string `req_header:"TE"`
}

func (headerConf *HeaderConfig) setDefaults() {
	if headerConf.Host == "" {
		headerConf.Host = "book.azal.az"
	}
	if headerConf.UserAgent == "" {
		headerConf.UserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/115.0"
	}
	if headerConf.Accept == "" {
		headerConf.Accept = "application/json, text/plain, */*"
	}
	if headerConf.AcceptLanguage == "" {
		headerConf.AcceptLanguage = "en-US,en;q=0.5"
	}
	if headerConf.AcceptEncoding == "" {
		headerConf.AcceptEncoding = "gzip, deflate, br"
	}
	if headerConf.XApplication == "" {
		headerConf.XApplication = "ibe"
	}
	if headerConf.XLocale == "" {
		headerConf.XLocale = "az"
	}
	if headerConf.Connection == "" {
		headerConf.Connection = "keep-alive"
	}
	if headerConf.SecFetchDest == "" {
		headerConf.SecFetchDest = "empty"
	}
	if headerConf.SecFetchMode == "" {
		headerConf.SecFetchMode = "cors"
	}
	if headerConf.SecFetchSite == "" {
		headerConf.SecFetchSite = "same-origin"
	}
	if headerConf.TE == "" {
		headerConf.TE = "trailers"
	}
}

func (headerConf *HeaderConfig) setToRequest(req *http.Request) {
	t := reflect.TypeOf(*headerConf)
	v := reflect.ValueOf(headerConf).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("req_header")
		value := v.Field(i).String()
		req.Header.Set(tag, value)
	}
}

type QueryConfig struct {
	Lang          string `req_query:"lang"`
	From          string `req_query:"from"`
	To            string `req_query:"to"`
	DepartureDate string `req_query:"departure_date"`
	TripType      string `req_query:"tripType"`
	AdultCount    string `req_query:"adult_count"`
	ChildCount    string `req_query:"child_count"`
	InfantCount   string `req_query:"infant_count"`
	IsStudent     string `req_query:"is_student"`
	Timestamp     string `req_query:"timestamp"`
	IsCitizen     string `req_query:"is_citizen"`
	Currency      string `req_query:"currency"`
	Theme         string `req_query:"theme"`
}

func (queryConf *QueryConfig) setDefaults() {
	if queryConf.Lang == "" {
		queryConf.Lang = "az"
	}
	if queryConf.TripType == "" {
		queryConf.TripType = "OW"
	}
	if queryConf.AdultCount == "" {
		queryConf.AdultCount = "1"
	}
	if queryConf.ChildCount == "" {
		queryConf.ChildCount = "0"
	}
	if queryConf.InfantCount == "" {
		queryConf.InfantCount = "0"
	}
	if queryConf.IsStudent == "" {
		queryConf.IsStudent = "0"
	}
	if queryConf.Timestamp == "" {
		queryConf.Timestamp = fmt.Sprintf("%d", time.Now().UnixNano()/int64(time.Millisecond))
	}
	if queryConf.IsCitizen == "" {
		queryConf.IsCitizen = "1"
	}
	if queryConf.Currency == "" {
		queryConf.Currency = "AZN"
	}
	if queryConf.Theme == "" {
		queryConf.Theme = "dark"
	}
}

func (queryConf *QueryConfig) setToRequest(req *http.Request) {
	q := req.URL.Query()
	t := reflect.TypeOf(*queryConf)
	v := reflect.ValueOf(queryConf).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("req_query")
		value := v.Field(i).String()
		q.Add(tag, value)
	}

	req.URL.RawQuery = q.Encode()
}

func handleErrorResponse(errorResponse *ErrorResponse) error {
	switch errorResponse.Error.Code {
	case "no.flights.available":
		return ErrorNoFlightsAvailable
	default:
		return fmt.Errorf("unknown error: %s", errorResponse.Error.Code)
	}
}

func sendRequest(queryConf *QueryConfig, headerConf *HeaderConfig) (*SuccessResponse, error) {
	req, err := http.NewRequest("GET", RequestURL, nil)
	if err != nil {
		return nil, err
	}

	headerConf.setToRequest(req)
	queryConf.setToRequest(req)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, err
	}
	errorResponseData := &ErrorResponse{}
	if err := json.Unmarshal(respBody, &errorResponseData); err != nil {
		return nil, err
	}
	if errorResponseData.Error.Code != "" {
		return nil, handleErrorResponse(errorResponseData)
	}
	successResponseData := &SuccessResponse{}
	if err := json.Unmarshal(respBody, &successResponseData); err != nil {
		return nil, err
	}
	return successResponseData, nil
}

func getUserInput() *UserInput {
	var (
		firstDate,
		lastDate,
		from,
		to,
		telegramBotKey,
		telegramChatID string
		repetInterval uint32
		userInput     = &UserInput{}
	)

	var rootCmd = &cobra.Command{
		Use:     "Azal Bot",
		Short:   "A CLI tool to find the flights",
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			first, err := time.Parse("2006-01-02T15:04:05", firstDate)
			if err != nil {
				first, err = time.Parse("2006-01-02", firstDate)
				if err != nil {
					fmt.Printf("Error: parsing FirstDate: %v\n", err)
					cmd.Help()
					os.Exit(1)
				}
			}
			last, err := time.Parse("2006-01-02T15:04:05", lastDate)
			if err != nil {
				last, err = time.Parse("2006-01-02", lastDate)
				if err != nil {
					fmt.Printf("Error: parsing LastDate: %v\n", err)
					cmd.Help()
					os.Exit(1)
				}
				last = last.AddDate(0, 0, 1)
				last = last.Add(-time.Second)
			}
			if first.After(last) || first.Equal(last) {
				fmt.Println("Error: first date should be before last date and they should not be equal")
				cmd.Help()
				os.Exit(1)
			}
			if repetInterval < 1 {
				fmt.Println("Error: repetInterval should be greater than 0")
				cmd.Help()
				os.Exit(1)
			}
			if len(from) > 5 || len(from) < 2 {
				fmt.Println("Error: from should be between 2 and 5 characters")
				cmd.Help()
				os.Exit(1)
			}
			if len(to) > 5 || len(to) < 2 {
				fmt.Println("Error: to should be between 2 and 5 characters")
				cmd.Help()
				os.Exit(1)
			}
			if telegramBotKey != "" {
				if telegramChatID == "" {
					fmt.Println("Error: telegramChatID is required if telegramBotKey is provided")
					cmd.Help()
					os.Exit(1)
				}
			}
			if telegramChatID != "" {
				if telegramBotKey == "" {
					fmt.Println("Error: telegramBotKey is required if telegramChatID is provided")
					cmd.Help()
					os.Exit(1)
				}
			}

			userInput.FirstDate = first
			userInput.LastDate = last
			userInput.From = from
			userInput.To = to
			userInput.TelegramBotKey = telegramBotKey
			userInput.TelegramChatID = telegramChatID
			userInput.RepetInterval = time.Duration(repetInterval) * time.Second
		},
	}

	rootCmd.Flags().StringVarP(&firstDate, "first-date", "i", "", "First date in format '2006-01-02T15:04:05'")
	rootCmd.Flags().StringVarP(&lastDate, "last-date", "l", "", "Last date in format '2006-01-02T15:04:05'")
	rootCmd.Flags().StringVarP(&from, "from", "f", "", "From where you want to fly (e.g. NAJ)")
	rootCmd.Flags().StringVarP(&to, "to", "t", "", "To where you want to fly (e.g. BAK)")
	rootCmd.Flags().StringVar(&telegramBotKey, "telegram-bot-key", "", "Telegram bot key")
	rootCmd.Flags().StringVar(&telegramChatID, "telegram-chat-id", "", "Telegram chat id")
	rootCmd.Flags().Uint32VarP(&repetInterval, "repet-interval", "r", 60, "Repetition interval in seconds")

	rootCmd.MarkFlagRequired("first-date")
	rootCmd.MarkFlagRequired("last-date")
	rootCmd.MarkFlagRequired("from")
	rootCmd.MarkFlagRequired("to")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rootCmd.Flags().Visit(func(flag *pflag.Flag) {
		switch flag.Name {
		case "version":
			os.Exit(0)
		case "help":
			os.Exit(0)
		}
	})

	return userInput
}

func startBot(botConfig *BotConfig, ifAvailable func(avialableFlights AvialableFlights) error) {
	queryConf := QueryConfig{
		From: botConfig.From,
		To:   botConfig.To,
	}
	queryConf.setDefaults()
	headerConf := HeaderConfig{}
	headerConf.setDefaults()

	for {
		avialableFlights := make(AvialableFlights)
		for _, day := range botConfig.days {
			queryConf.DepartureDate = day
			data, err := sendRequest(&queryConf, &headerConf)
			if err != nil {
				if err == ErrorNoFlightsAvailable {
					log.Println(Colored(Colors.Yellow, "No flights available for ", day))
					continue
				}
				log.Println(Colored(Colors.Red, err.Error()))
				continue
			}

			if len(data.Warnings) > 0 {
				log.Println(Colored(Colors.Yellow, "No flights available for ", day))
				continue
			}
			for _, option := range data.Search.OptionSets[0].Options {
				departureDate := option.Route.DepartureDate
				if (departureDate.After(botConfig.FirstDate) || departureDate.Equal(botConfig.FirstDate)) &&
					(departureDate.Before(botConfig.LastDate) || departureDate.Equal(botConfig.LastDate)) {

					avialableFlights[day] = append(avialableFlights[day], departureDate.Time)
					log.Println(Colored(Colors.Green, "Flight available for ", departureDate))
				} else {
					log.Println(Colored(Colors.Yellow, "No flights available for ", departureDate))
				}
			}
		}
		if err := ifAvailable(avialableFlights); err != nil {
			log.Println(Colored(Colors.Red, "Error: ", err.Error()))
		}
		time.Sleep(botConfig.RepetInterval)
	}
}

func main() {
	userInput := getUserInput()
	botConfig := &BotConfig{
		FirstDate:     userInput.FirstDate,
		LastDate:      userInput.LastDate,
		From:          userInput.From,
		To:            userInput.To,
		RepetInterval: userInput.RepetInterval,
	}
	for current := userInput.FirstDate; !current.After(userInput.LastDate); current = current.AddDate(0, 0, 1) {
		botConfig.days = append(botConfig.days, current.Format("2006-01-02"))
	}

	ifAvailableFunc := func(avialableFlights AvialableFlights) error { return nil }
	if userInput.TelegramBotKey != "" {
		telegramRequest := &TelegramRequest{
			Client: &http.Client{},
			BotKey: userInput.TelegramBotKey,
			ChatID: userInput.TelegramChatID,
		}
		if err := telegramRequest.sendTelegramStartNotification(botConfig); err != nil {
			log.Println(Colored(Colors.Red, err.Error()))
		}
		ifAvailableFunc = func(avialableFlights AvialableFlights) error {
			if len(avialableFlights) > 0 {
				return telegramRequest.sendTelegramFlightNotification(avialableFlights)
			}
			return nil
		}
	}

	startBot(
		botConfig,
		ifAvailableFunc,
	)
}
