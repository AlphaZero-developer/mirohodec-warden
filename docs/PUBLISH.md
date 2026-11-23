# Публикация на GitHub

1. Сначала перевыпустите токен Telegram-бота и ключ OpenRouter, которые присутствовали в исходном архиве. Новый публичный ZIP их не содержит.
2. Извлеките ZIP в отдельную папку. **Не копируйте `.git`, конфиги и данные из старой директории проекта.**
3. Проверьте публичные файлы:

   ```bash
   python3 scripts/check_public.py
   go test ./...
   go vet ./...
   go build ./cmd/warden
   ```

4. Создайте пустой репозиторий на GitHub, например `mirohodec-warden`, и выполните:

   ```bash
   git init
   git branch -M main
   git add .
   git status --short
   git commit -m "Initial public backend showcase"
   git remote add origin https://github.com/AlphaZero-developer/mirohodec-warden.git
   git push -u origin main
   ```

   Если при создании репозитория GitHub уже добавил README или лицензию, согласуйте истории перед push. Адрес выше предполагает, что вы выбрали именно это имя.

5. В настройках репозитория добавьте темы `golang`, `telegram-bot`, `backend`, `postgresql`, `moderation`, `websocket`, `ai-integration`, `portfolio`.
6. Для резюме используйте `docs/CASE_STUDY.md`, без недоказанных чисел пользователей, выручки или времени безотказной работы.

Папка не содержит лицензии, предоставляющей третьим лицам права на повторное использование. Если исходный проект создавался по договору с заказчиком, отдельно проверьте принадлежность прав на публикацию кода.
