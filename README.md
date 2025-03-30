Такая же инструкция вниза файла cmd/main.go



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
заходим в файл окружения bot/cmd/ENV.env и вносим туда имя параметры бота 

токен 
имя команды 
имя канала
имя бота 
адрес сервера 

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

