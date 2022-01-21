package main

import (
	"fmt"
	tcp "github.com/ilinovalex86/tcpserver"
	web "github.com/ilinovalex86/tcpserverweb"
	"log"
	"net/http"
)

const tcpServer = "ipAddress:port"
const webServer = "ipAddress:port"

//Привязывает tcp клиента к web клиенту
func auth(w http.ResponseWriter, r *http.Request) {
	if _, valid := web.Valid(r); valid {
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		tcpId := r.FormValue("clientId")
		webId := web.GeneratorId()
		cookie := http.Cookie{Name: "SessionId", Value: webId}
		http.SetCookie(w, &cookie)
		web.Clients.Store(webId, tcpId)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}

//Предлагает выбрать tcp клиента для подключения
func login(w http.ResponseWriter, r *http.Request) {
	if _, valid := web.Valid(r); valid {
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		var webR web.Response
		webR.FreeTcpClients = tcp.Clients.Available()
		if len(webR.FreeTcpClients) == 0 {
			webR.Error = "Нет доступных компьютеров."
			web.ToBrowser(w, "./html/error.html", webR)
		} else {
			web.ToBrowser(w, "./html/login.html", webR)
		}

	}
}

//Принимает запросы для tcp клиента и выводит ответ
func index(w http.ResponseWriter, r *http.Request) {
	if id, valid := web.Valid(r); valid {
		var webR web.Response
		webR.User = tcp.Clients.User(id)
		webR.WebServer = webServer
		path := r.FormValue("path")
		res, err := tcp.Clients.Dir(id, path)
		if err != nil {
			if fmt.Sprint(err) == "error conn" {
				htmlFile := "./html/error.html"
				webR.Error = "Компьютер не подключен."
				web.ToBrowser(w, htmlFile, webR)
			}
			http.Redirect(w, r, "/", http.StatusFound)
		}
		webR.Menu = res["nav"]
		webR.Dirs = res["dirs"]
		webR.Files = res["files"]
		webR.Sep = tcp.Clients.Sep(id)
		htmlFile := "./html/dir.html"
		web.ToBrowser(w, htmlFile, webR)
	} else {
		web.ClientClear(w, r)
	}
}

//Выдает файл на скачивание
func sendFile(w http.ResponseWriter, r *http.Request) {
	if id, valid := web.Valid(r); valid {
		path := r.FormValue("path")
		name := r.FormValue("name")
		res, err := tcp.Clients.File(id, path)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusFound)
		}
		w.Header().Set("Content-Disposition", "attachment; filename="+name)
		w.Header().Set("Content-Type", "application/octet-stream")
		http.ServeFile(w, r, res)
	} else {
		web.ClientClear(w, r)
	}
}

//Разлогинивает web клиента
func logout(w http.ResponseWriter, r *http.Request) {
	web.ClientClear(w, r)
}

func main() {
	go tcp.Listen(tcpServer)
	mux := http.NewServeMux()
	mux.HandleFunc("/", index)
	mux.HandleFunc("/file", sendFile)
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/auth", auth)
	mux.HandleFunc("/logout", logout)
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))
	err := http.ListenAndServe(webServer, mux)
	log.Fatal(err)
}
