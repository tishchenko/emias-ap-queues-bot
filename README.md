# emias-ap-queues-bot

Telegram bot для мониторинга состояния очередей сераиса АП.Агрегаты.

В случае роста очередей выше допустимых значений, бот будет выдавать соответствующее оповещение.

Пример конфигурационного файла config.json (должен располагаться в одном каталоге с исполняемым фпйлом):
{
  "token": "585604919:AAG_wqdpDE5zg3bGJznhl0ZTVT2NqOpwyHs",
  "proxy": {
    "ip": "xxx.xxx.xxx.xxx",
    "port": "xxxx",
    "user": "xxx",
    "password": "xxx"
  },
  "modelFileName": "../m.js",
  "alarmLogic": {
    "normalQueues": {
      "APPOINTMENT": 969,
      "AR_SCHEDULE_UPDATED": 969,
      "SELF_APPOINTMENT": 969,
      "UNMET_DEMAND": 969
    },
    "exceptionQueues": {
      "APPOINTMENT": 69,
      "AR_SCHEDULE_UPDATED": 96,
      "SELF_APPOINTMENT": 69,
      "UNMET_DEMAND": 69
    },
    "pollInterval": 3600
  }
}

pollInterval - интервал в секундах, с которым будет опрашиваться файл с данными modelFileName
alarmLogic - блок, который для каждой очереди указывает минимальную дельту в показаниях зв смежные интервалы, при привышении которой необходимо создавать оповещение
proxy - параметры прокси для Telegram (если прокси использовать нет необходимости, то блок может быть удалён из файла config.json