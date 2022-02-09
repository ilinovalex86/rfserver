package main

import (
	"fmt"
	tcp "github.com/ilinovalex86/tcpserver"
	web "github.com/ilinovalex86/tcpserverweb"
	"log"
	"net/http"
)

const tcpServer = "ipAddress:port"
const streamServer = "ipAddress:port"
const webServer = "ipAddress:port"
const boundaryWord = "MJPEGBOUNDARY"
const headerf = "\r\n" +
	"--" + boundaryWord + "\r\n" +
	"Content-Type: image/jpeg\r\n" +
	"Content-Length: %d\r\n" +
	"X-Timestamp: 0.000000\r\n" +
	"\r\n"

//Привязывает tcp клиента к web клиенту
func auth(w http.ResponseWriter, r *http.Request) {
	if _, valid := web.Valid(r); valid {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	tcpId := r.FormValue("clientId")
	webId := web.GeneratorId()
	cookie := http.Cookie{Name: "SessionId", Value: webId}
	http.SetCookie(w, &cookie)
	web.Clients.Store(webId, tcpId)
	http.Redirect(w, r, "/", http.StatusFound)
}

//Предлагает выбрать tcp клиента для подключения
func login(w http.ResponseWriter, r *http.Request) {
	if _, valid := web.Valid(r); valid {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	var webR web.Response
	webR.FreeTcpClients = tcp.Clients.Available()
	if len(webR.FreeTcpClients) == 0 {
		webR.Error = "Нет доступных компьютеров."
		web.ToBrowser(w, "./html/error.html", webR)
	} else {
		web.ToBrowser(w, "./html/login.html", webR)
	}
}

//Принимает запросы для tcp клиента и выводит ответ
func index(w http.ResponseWriter, r *http.Request) {
	id, valid := web.Valid(r)
	if !valid {
		web.ClientClear(w, r)
		return
	}
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
		return
	}
	webR.Menu = res["nav"]
	webR.Dirs = res["dirs"]
	webR.Files = res["files"]
	webR.Sep = tcp.Clients.Sep(id)
	htmlFile := "./html/dir.html"
	web.ToBrowser(w, htmlFile, webR)
}

//Выдает файл на скачивание
func sendFile(w http.ResponseWriter, r *http.Request) {
	id, valid := web.Valid(r)
	if !valid {
		web.ClientClear(w, r)
		return
	}
	path := r.FormValue("path")
	name := r.FormValue("name")
	res, err := tcp.Clients.File(id, path)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+name)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, res)
}

//Разлогинивает web клиента
func logout(w http.ResponseWriter, r *http.Request) {
	web.ClientClear(w, r)
}

// Запускает стрим рабочего стола клиента
func stream(w http.ResponseWriter, r *http.Request) {
	id, valid := web.Valid(r)
	if !valid {
		web.ClientClear(w, r)
		return
	}
	cookie, err := r.Cookie("SessionId")
	webId := cookie.Value
	s, err := tcp.Clients.Stream(id, webId)
	if err != nil {
		var webR web.Response
		webR.User = tcp.Clients.User(id)
		htmlFile := "./html/error.html"
		webR.Error = fmt.Sprint(err)
		web.ToBrowser(w, htmlFile, webR)
		return
	}
	imgIndex := 0
	w.Header().Add("Content-Type", "multipart/x-mixed-replace;boundary="+boundaryWord)
	for {
		i, img, err := s.Next(imgIndex)
		if err != nil {
			break
		}
		header := fmt.Sprintf(headerf, len(img))
		frame := make([]byte, len(img)+len(header))
		copy(frame, header)
		copy(frame[len(header):], img)
		_, err = w.Write(frame)
		if err != nil {
			break
		}
		imgIndex = i
	}
	s.Remove(webId)
}

func main() {
	go tcp.TcpServer(tcpServer)
	go tcp.StreamServer(streamServer)
	mux := http.NewServeMux()
	mux.HandleFunc("/", index)
	mux.HandleFunc("/stream", stream)
	mux.HandleFunc("/file", sendFile)
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/auth", auth)
	mux.HandleFunc("/logout", logout)
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))
	err := http.ListenAndServe(webServer, mux)
	log.Fatal(err)
}
