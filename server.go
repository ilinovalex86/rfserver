package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	ie "github.com/ilinovalex86/inputevent"
	tcp "github.com/ilinovalex86/tcpserver"
	web "github.com/ilinovalex86/tcpserverweb"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
)

const boundaryWord = "MJPEGBOUNDARY"
const headerf = "\r\n" +
	"--" + boundaryWord + "\r\n" +
	"Content-Type: image/jpeg\r\n" +
	"Content-Length: %d\r\n" +
	"X-Timestamp: 0.000000\r\n" +
	"\r\n"

// Отправляет файл клиенту
func sendFile(w http.ResponseWriter, r *http.Request) {
	id, valid := web.Valid(r)
	if !valid {
		web.ClientClear(w, r)
		return
	}
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	buffer := make([]byte, handler.Size)
	reader := bufio.NewReader(file)
	_, err = reader.Read(buffer)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	file.Close()
	err = tcp.Clients.FileToClient(id, &buffer, handler.Filename, handler.Size)
	if err != nil {
		fmt.Println(err)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

//Привязывает web клиента к tcp клиенту
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
		return
	}
	web.ToBrowser(w, "./html/login.html", webR)
}

//Принимает файлы и папки клиента
func index(w http.ResponseWriter, r *http.Request) {
	id, valid := web.Valid(r)
	if !valid {
		web.ClientClear(w, r)
		return
	}
	var webR web.Response
	webR.User = id
	webR.WebServer = tcp.Conf.WebServer
	path := r.FormValue("path")
	res, sep, err := tcp.Clients.Dir(id, path)
	if err != nil {
		htmlFile := "./html/error.html"
		webR.Error = fmt.Sprint(err)
		web.ToBrowser(w, htmlFile, webR)
		return
	}
	webR.Menu = res["nav"]
	webR.Dirs = res["dirs"]
	webR.Files = res["files"]
	webR.Sep = sep
	htmlFile := "./html/dir.html"
	web.ToBrowser(w, htmlFile, webR)
}

//Скачивает файл с клиента
func getFile(w http.ResponseWriter, r *http.Request) {
	id, valid := web.Valid(r)
	if !valid {
		web.ClientClear(w, r)
		return
	}
	path := r.FormValue("path")
	name := r.FormValue("name")
	path, err := tcp.Clients.FileFromClient(id, path)
	if err != nil {
		var webR web.Response
		webR.User = id
		htmlFile := "./html/error.html"
		webR.Error = fmt.Sprint(err)
		web.ToBrowser(w, htmlFile, webR)
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+name)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, path)
	os.Remove(path)
}

//Выдает файл на скачивание
func client(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=client.exe")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, "client.exe")
}

// Отдает стрим рабочего стола клиента
func stream(w http.ResponseWriter, r *http.Request) {
	id, valid := web.Valid(r)
	if !valid {
		web.ClientClear(w, r)
		return
	}
	s, err := tcp.Clients.StreamGet(id)
	if err != nil {
		fmt.Println(err)
		return
	}
	imgIndex := 0
	w.Header().Add("Content-Type", "multipart/x-mixed-replace;boundary="+boundaryWord)
	for {
		t := time.Now()
		i, img, err := s.ImgNext(imgIndex)
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
		fmt.Println("server: ", time.Since(t).Milliseconds(), imgIndex)
	}
	cookie, _ := r.Cookie("SessionId")
	webId := cookie.Value
	s.WebClientRemove(webId)
}

// Запускает стрим рабочего стола клиента
func remote(w http.ResponseWriter, r *http.Request) {
	id, valid := web.Valid(r)
	if !valid {
		web.ClientClear(w, r)
		return
	}
	cookie, _ := r.Cookie("SessionId")
	webId := cookie.Value
	stream, err := tcp.Clients.StreamStart(id, webId)
	var webR web.Response
	if err != nil {
		webR.User = id
		htmlFile := "./html/error.html"
		webR.Error = fmt.Sprint(err)
		web.ToBrowser(w, htmlFile, webR)
		return
	}
	webR.WebServer = tcp.Conf.WebServer
	webR.ScreenSizeX = stream.ScreenSizeX
	webR.ScreenSizeY = stream.ScreenSizeY
	var files []string
	files = append(files, "html/remote.html")
	html, err := template.ParseFiles(files...)
	if err != nil {
		log.Fatal(err)
	}
	err = html.Execute(w, webR)
	if err != nil {
		log.Fatal(err)
	}
}

func event(w http.ResponseWriter, r *http.Request) {
	id, valid := web.Valid(r)
	if !valid {
		web.ClientClear(w, r)
		return
	}
	decoder := json.NewDecoder(r.Body)
	var event ie.Event
	err := decoder.Decode(&event)
	if err != nil {
		fmt.Fprint(w, "decode error")
		return
	}
	s, err := tcp.Clients.StreamGet(id)
	if err != nil {
		fmt.Fprint(w, "stream error")
		return
	}
	fmt.Printf("%#v \n", event)
	s.EventAdd(event)
	fmt.Fprint(w, "done")
}

//Разлогинивает web клиента
func logout(w http.ResponseWriter, r *http.Request) {
	web.ClientClear(w, r)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", index)
	mux.HandleFunc("/stream", stream)
	mux.HandleFunc("/getFile", getFile)
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/auth", auth)
	mux.HandleFunc("/client", client)
	mux.HandleFunc("/logout", logout)
	mux.HandleFunc("/event", event)
	mux.HandleFunc("/remote", remote)
	mux.HandleFunc("/sendFile", sendFile)
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))
	err := http.ListenAndServe(tcp.Conf.WebServerListner, mux)
	log.Fatal(err)
}
