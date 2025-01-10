# Сервис для генерации комментариев в gitlab от sonarqube
## Сборка
docker build . -t go
## Запуск
docker run --name go   -p 8181:8080   --env-file env.list   -d  go
## env.list
GITLAB_URL=https://Ссылка_на_гит

GITLAB_TOKEN="Токен с доступом API"

## использование
Прописать адрес:  
http://[IP]:8181/webhook  
в вебхуках проекта сонар
