package main

import (
	"bot/domain"
	ram "bot/repo/ramStorage"
	//tntRepo "bot/repo/tarantool"
	//tt "github.com/tarantool/go-tarantool"

	//tntRepo "bot/repo/tarantool"
	svc "bot/usecases/service"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/rs/zerolog"
	//tt "github.com/tarantool/go-tarantool"
	"log"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"time"
)

//github.com/mattermost/mattermost-server/v6/model

// Config хранит параметры подключения к Mattermost.
type Config struct {
	MattermostUserName string
	MattermostTeamName string
	MattermostToken    string
	MattermostChannel  string
	MattermostServer   *url.URL
}

// loadConfig загружает конфигурацию из переменных окружения.
func loadConfig() Config {
	if err := godotenv.Load("ENV.env"); err != nil {
		log.Printf("Не удалось загрузить .env-файл: %v", err)
	}
	serverRaw := os.Getenv("MM_SERVER")

	serverURL, err := url.Parse(serverRaw) //url.Parse("http://localhost:8065") //url.Parse(os.Getenv("MM_SERVER"))
	if err != nil {
		log.Printf("Ошибка парсинга URL из переменной окружения MM_SERVER: %v", err)
	}

	return Config{
		MattermostUserName: os.Getenv("MM_USERNAME"), //"PollBot",                    //os.Getenv("MM_USERNAME"),
		MattermostTeamName: os.Getenv("MM_TEAM"),     //"mmm",                        //os.Getenv("MM_TEAM"),
		MattermostToken:    os.Getenv("MM_TOKEN"),    //"37345hchoigwjnahw4m9pmfgbo", //os.Getenv("MM_TOKEN"),
		MattermostChannel:  os.Getenv("MM_CHANNEL"),  //"town-square",                // os.Getenv("MM_CHANNEL"),
		MattermostServer:   serverURL,
	}
}

// Application содержит зависимости и объекты для работы бота.
type Application struct {
	config                    Config
	logger                    zerolog.Logger
	mattermostClient          *model.Client4
	mattermostWebSocketClient *model.WebSocketClient
	mattermostUser            *model.User
	mattermostChannel         *model.Channel
	mattermostTeam            *model.Team

	service *svc.Service
}

// service *service.Service
func NewApplication(service *svc.Service, config Config, logger zerolog.Logger) *Application {
	/*logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC822,
	}).With().Timestamp().Logger()

	config := loadConfig()
	*/
	return &Application{
		config:  config,
		logger:  logger,
		service: service,
	}
}

// sendMsgToChannel отправляет сообщение в указанный канал Mattermost.
func (app *Application) sendMsgToChannel(message, replyToID string) {
	post := &model.Post{
		ChannelId: app.mattermostChannel.Id,
		Message:   message,
		RootId:    replyToID, // если ответ в треде
	}
	if _, _, err := app.mattermostClient.CreatePost(post); err != nil {
		app.logger.Error().Err(err).Str("replyToID", replyToID).Msg("Не удалось отправить сообщение")
	}
}

func parseCommand(input string) []string {
	re := regexp.MustCompile(`"([^"]+)"|(\S+)`)
	matches := re.FindAllStringSubmatch(input, -1)
	var args []string
	for _, match := range matches {
		if match[1] != "" {
			args = append(args, match[1])
		} else {
			args = append(args, match[2])
		}
	}
	return args
}

// validateCommand принимает разобранные аргументы и возвращает номер команды:
// 1 - /createPoll, 2 - /vote, 3 - /result, 4 - /endPoll, 5 - /delete.
// Если синтаксис не соответствует ни одной команде или аргументы заданы неверно, функция возвращает 0.
func validateCommand(args []string) int {
	if len(args) == 0 {
		return 0
	}

	switch args[0] {
	case "#createPoll":
		// Минимальное количество аргументов: команда, название опроса, количество опций и хотя бы одна опция.
		if len(args) < 4 {
			return 0
		}
		// args[2] должно содержать число, равное количеству опций.
		n, err := strconv.Atoi(args[2])
		if err != nil || n < 1 {
			return 0
		}
		// Ожидаем ровно n опций после первых трёх аргументов.
		if len(args) != 3+n {
			return 0
		}
		return 1

	case "#vote":
		// Ожидаем ровно: команда, id опроса, вариант ответа.
		if len(args) != 3 {
			return 0
		}
		return 2

	case "#result":
		// Ожидаем ровно: команда, id опроса.
		if len(args) != 2 {
			return 0
		}
		return 3

	case "#endPoll":
		// Ожидаем ровно: команда, id опроса.
		if len(args) != 2 {
			return 0
		}
		return 4

	case "#delete":
		// Ожидаем ровно: команда, id опроса.
		if len(args) != 2 {
			return 0
		}
		return 5

	default:
		return 0
	}
}

/*
/createPoll <PollName> <Количество опций(n)> <Option1_text> ... <OptionN_text> (1)
/vote <Poll id> <Option text> (2)
/result <Poll id> (3)
/endPoll <Poll id> (4)
/delete <Poll id> (5)
*/

// handleWebSocketEvent обрабатывает входящие события по WebSocket.
func (app *Application) handleWebSocketEvent(event *model.WebSocketEvent) {
	var dataToProcess []string

	// Фильтруем события: нас интересуют только сообщения в нужном канале.
	if event.GetBroadcast().ChannelId != app.mattermostChannel.Id {
		return
	}
	if event.EventType() != model.WebsocketEventPosted {
		return
	}

	// Десериализуем данные сообщения.
	post := &model.Post{}
	if err := json.Unmarshal([]byte(event.GetData()["post"].(string)), post); err != nil {
		app.logger.Error().Err(err).Msg("Ошибка парсинга данных поста")
		return
	}

	// Игнорируем сообщения, отправленные самим ботом.
	if post.UserId == app.mattermostUser.Id {
		return
	}

	// Формируем строку с данными полученного сообщения.

	dataToProcess = parseCommand(post.Message)

	num := validateCommand(dataToProcess)

	switch num {
	case 0:
		app.sendMsgToChannel("Invalid input. Check correction: \n/createPoll <PollName> <Количество опций(n)> <Option1_text> ... <OptionN_text> \n/vote <Poll id> <Option text> \n/result <Poll id> \n/endPoll <Poll id> \n/delete <Poll id> ", post.Id)
	case 1:
		// Ожидаемый формат:
		// dataToProcess[0] : "/createPoll"
		// dataToProcess[1] : PollName (описание опроса)
		// dataToProcess[2] : количество опций (n) в виде строки
		// dataToProcess[3] ... dataToProcess[3+n-1] : текст каждой опции

		// Преобразуем количество опций в число
		numOptions, err := strconv.Atoi(dataToProcess[2])
		if err != nil || numOptions < 1 || len(dataToProcess) != 3+numOptions {
			app.sendMsgToChannel("Invalid input for /createPoll command", post.Id)
			break
		}

		newPollID := uuid.New().String()
		var options []domain.Option
		for i := 0; i < numOptions; i++ {
			opt := domain.Option{
				OptID: uuid.New().String(),
				Text:  dataToProcess[3+i],
				Votes: 0,
			}
			options = append(options, opt)
		}
		poll, err := app.service.CreatePoll(newPollID, dataToProcess[1], options)
		if err != nil {
			app.sendMsgToChannel("Error creating poll: "+err.Error(), post.Id)
		} else {
			response := "Poll created successfully!\nPoll ID: " + poll.ID + "\nOptions:\n"
			for _, opt := range poll.Options {
				response += "- " + opt.Text + "\n"
			}
			app.sendMsgToChannel(response, post.Id)
		}
	case 2:
		// Ожидаемый формат: /vote <Poll id> <Option text>
		if len(dataToProcess) != 3 {
			app.sendMsgToChannel("Invalid input for /vote command. Correct usage: /vote <Poll id> <Option text>", post.Id)
			break
		}
		pollID := dataToProcess[1]
		optionText := dataToProcess[2]

		// Получаем опрос по ID
		poll, err := app.service.GetPoll(pollID)
		if err != nil {
			app.sendMsgToChannel("Poll not found: "+err.Error(), post.Id)
			break
		}

		// Проверяем, что опрос активен
		if !poll.Active {
			app.sendMsgToChannel("This poll is closed and no longer accepts votes.", post.Id)
			break
		}

		// Если необходимо ограничить голосование одним голосом на пользователя,
		// здесь можно проверить, не голосовал ли уже этот пользователь.
		// Например, можно хранить список userID, отдавших голос в опросе.

		var selectedOptionID string
		for _, opt := range poll.Options {
			if opt.Text == optionText {
				selectedOptionID = opt.OptID
				break
			}
		}
		if selectedOptionID == "" {
			app.sendMsgToChannel("Option not found in poll", post.Id)
			break
		}

		// Вызываем бизнес-логику голосования
		err = app.service.VotePoll(poll, selectedOptionID)
		if err != nil {
			app.sendMsgToChannel("Error voting: "+err.Error(), post.Id)
		} else {
			app.sendMsgToChannel("Vote registered successfully!", post.Id)
			// Опционально, можно сразу вернуть обновленные результаты опроса
		}
	case 3:
		// Ожидаемый формат: /result <Poll id>
		if len(dataToProcess) != 2 {
			app.sendMsgToChannel("Invalid input for /result command. Correct usage: /result <Poll id>", post.Id)
			break
		}
		pollID := dataToProcess[1]
		options, err := app.service.GetPollOptions(pollID)
		if err != nil {
			app.sendMsgToChannel("Error retrieving poll: "+err.Error(), post.Id)
			break
		}
		if options == nil || len(*options) == 0 {
			app.sendMsgToChannel("Poll not found or no options available", post.Id)
			break
		}
		response := "Poll results for poll " + pollID + ":\n"
		for _, opt := range *options {
			response += fmt.Sprintf("Option: %s, Votes: %d\n", opt.Text, opt.Votes)
		}
		app.sendMsgToChannel(response, post.Id)
	case 4:
		// Ожидаемый формат: /endPoll <Poll id>
		if len(dataToProcess) != 2 {
			app.sendMsgToChannel("Invalid input for /endPoll command. Correct usage: /endPoll <Poll id>", post.Id)
			break
		}
		pollID := dataToProcess[1]
		err := app.service.EndPoll(pollID)
		if err != nil {
			app.sendMsgToChannel("Error ending poll: "+err.Error(), post.Id)
			break
		}
		poll, err := app.service.GetPoll(pollID)
		if err != nil {
			app.sendMsgToChannel("Poll ended but error retrieving results: "+err.Error(), post.Id)
			break
		}
		// Формируем сообщение с итоговыми результатами
		response := "Poll ended successfully! Final results for poll " + pollID + ":\n"
		for _, opt := range poll.Options {
			response += fmt.Sprintf("Option: %s, Votes: %d\n", opt.Text, opt.Votes)
		}
		app.sendMsgToChannel(response, post.Id)
	case 5:
		// Ожидаемый формат: /delete <Poll id>
		if len(dataToProcess) != 2 {
			app.sendMsgToChannel("Invalid input for /delete command. Correct usage: /delete <Poll id>", post.Id)
			break
		}
		pollID := dataToProcess[1]
		err := app.service.DeletePoll(pollID)
		if err != nil {
			app.sendMsgToChannel("Error deleting poll: "+err.Error(), post.Id)
		} else {
			app.sendMsgToChannel("Poll deleted successfully!", post.Id)
		}
	default:
		app.sendMsgToChannel("China number 1", post.Id)
	}

	//receivedData := fmt.Sprintf("Бот получил сообщение: %s", post.Message)
	//app.logger.Info().Msg(receivedData)

	// Отправляем ответное сообщение в тот же тред.
	// app.sendMsgToChannel(receivedData, post.Id)
}

// listenToEvents запускает цикл получения событий по WebSocket.

// ! ошибка сокет не подключается
func (app *Application) listenToEvents() {
	var err error
	for {
		// Формируем URL для WebSocket (используем wss:// для TLS)
		wsURL := fmt.Sprintf("ws://%s%s", app.config.MattermostServer.Host, app.config.MattermostServer.Path)
		app.mattermostWebSocketClient, err = model.NewWebSocketClient4(wsURL, app.mattermostClient.AuthToken)
		if err != nil {
			app.logger.Error().Err(err).Msg("Ошибка подключения к WebSocket, повтор через 5 секунд")
			time.Sleep(5 * time.Second)
			continue
		}
		app.logger.Info().Msg("WebSocket подключён")
		app.mattermostWebSocketClient.Listen()

		// Читаем события из канала
		for event := range app.mattermostWebSocketClient.EventChannel {
			// Каждое событие обрабатываем в отдельной горутине
			go app.handleWebSocketEvent(event)
		}
	}
}

// setupGracefulShutdown обеспечивает корректное завершение работы бота.
func setupGracefulShutdown(app *Application) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if app.mattermostWebSocketClient != nil {
				app.logger.Info().Msg("Закрытие WebSocket соединения")
				app.mattermostWebSocketClient.Close()
			}
			app.logger.Info().Msg("Выход из приложения")
			os.Exit(0)
		}
	}()
}

func main() {

	/*
		ttStorage, err := tntRepo.NewTarantoolStorage("127.0.0.1:3301", tt.Opts{
			User: "admin",       // как в docker-compose
			Pass: "examplepass", // см. TARANTOOL_USER_PASSWORD
			// Timeout: time.Second, // если нужно задать таймаут
		})
		if err != nil {
			log.Fatalf("Failed to connect to Tarantool: %s", err)
		}
	*/

	logger := zerolog.New(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC822,
	}).With().Timestamp().Logger()

	config := loadConfig()
	logger.Info().Msgf("Конфигурация загружена: %+v", config)

	rs := ram.NewRamStorage()
	sv := svc.NewService(rs)
	//sv := svc.NewService(ttStorage)
	app := NewApplication(sv, config, logger)

	setupGracefulShutdown(app)

	app.mattermostClient = model.NewAPIv4Client(config.MattermostServer.String())
	app.mattermostClient.SetToken(config.MattermostToken)

	// Получаем данные бота
	user, _, err := app.mattermostClient.GetUser("me", "")
	if err != nil {
		logger.Fatal().Err(err).Msg("Не удалось выполнить вход")
	}
	app.mattermostUser = user
	logger.Info().Msg("Бот авторизован в Mattermost")

	// Получаем информацию о команде.
	team, _, err := app.mattermostClient.GetTeamByName(config.MattermostTeamName, "")
	if err != nil {
		logger.Fatal().Err(err).Msg("Не удалось найти команду")
	}
	app.mattermostTeam = team

	// Получаем информацию о канале.
	channel, _, err := app.mattermostClient.GetChannelByName(config.MattermostChannel, team.Id, "")
	if err != nil {
		logger.Fatal().Err(err).Msg("Не удалось найти канал")
	}
	app.mattermostChannel = channel

	app.sendMsgToChannel("Привет, я активен и жду входящие сообщения", "")

	app.listenToEvents()
}

/*
[Пользователь]
     ↓ (отправка сообщения)
[Сервер Mattermost]
     ↓ (рассылка событий через WebSocket)
[Бот (с токеном)] ↔ [Ваш код-обработчик + in-memory БД]
     ↑ (отправка ответа через API)
[Сервер Mattermost]
     ↓ (отображение сообщения в чате)
[Пользователь]
*/

/*
token: 37345hchoigwjnahw4m9pmfgbo
name: PollBot
display name: display
*/

//todo
// починить окружение ENV.env (v)
// mm-ser -> data(string) *parse* -> usecases -> обработать строку получить результат вернуть ответ (v)
// узнать можно ли как то красиво опрос оформить
// поменять команды с / на  # (v)
// tt db
// раскидать сущности по папкам : логгер , приложение
// ENV.go сделать

/*
/createPoll <PollName> <Количество опций(n)> <Option1_text> ... <OptionN_text> (1)
/vote <Poll id> <Option text> (2)
/result <Poll id> (3)
/endPoll <Poll id> (4)
/delete <Poll id> (5)
*/

// запускаем docker daemon на локальной машине
// поднимаем mm-server (у меня локально на порте :8065) для этого /bot/mm-server : docker-compose up -d
// заходим на localhost:8065 регистрируемся и заходим в канал Town Square
// заходим в настройки в mattermost левый верхний угол
// integrations
// bot accaunt
// add bot accaunt создаем бота указывая его имя и получая его токен
// в чате добавляем его
// поднимаем tarantool /bot/tt : docker-compose up -d tarantool
// bot/cmd : go run main.go (go get <все зависимости>)

// далее играемся с ботом (важно учесть кавычки при запросах )
/*
пример
#createPoll "Best Programming Language" 3 "Go" "Python" "JavaScript"
получили uuid в ответ
далее запросы в формате:
//  #createPoll "Best Programming Language" 3 "Go" "Python" "JavaScript"
// #vote "<uuid опроса>" "варант"
// #result "<uuid опроса>"
// #endPoll "<uuid опроса>"
// #deletePoll "<uuid опроса>"

*/
