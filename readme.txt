Remote File Server это клиент-серверное приложение, предоставляющее
 доступ к файлам удаленных компьютеров через браузер.
Состоит из клиента и сервера.

Client - кроссплатформенный, определяет текущего пользователя и его домашнюю папку.
Получает путь от сервера и отправляет содержимое каталога или файл.
В ОС Windows Client расчитан на простых пользователей, поэтому:
	Подменяет путь к домашней папке пользователя на "Компьютер".
	Не отображает часть папок и файлы в домашней папке.
	Если нужно получить доступ к каталогу уровнем выше, например к диску "С".
	Пропишите в get запросе "C:" или "C:\".
В ОС Linux ограничений нет. Имеет доступ соотвествующий пользователю.
Протестирован на Windows10, Linux Ubuntu/Debian.

Server - серверная часть состоит из веб сервера и tcp сервера.
Сохраняет информацию о tcp и web клиентах в json файлах.
Привязывает web клиентов к tcp клиенту по средствам сессий.
Восстанавливает иформацию о tcp клиентах и сессии web клиентов при перезапуске Server'a.
	 
