package main

import (
	"flag"
	"github.com/goland-amaurybrisou/couchsport/api/handlers"
	"github.com/goland-amaurybrisou/couchsport/api/stores"
	"github.com/goland-amaurybrisou/couchsport/api/validators"
	"github.com/goland-amaurybrisou/couchsport/config"
	"github.com/goland-amaurybrisou/couchsport/localizer"
	"github.com/goland-amaurybrisou/couchsport/server"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	env := flag.String("env", "dev", "select environment config file to use (will load config.[env].json")
	populate := flag.Bool("populate", false, "inject fixtures in database")
	flag.Parse()

	c := config.Load(*env)
	c.Populate = *populate

	localizer := localizer.NewLocalizer(c.Localizer.LanguageFiles)

	srv := server.NewInstance(c)

	storeFactory := stores.NewStoreFactory(srv.Db, localizer, *c)
	storeFactory.Init(c.Populate)

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	if *env == "dev" {
		log.Println("enable WebSocket All Origins")
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	}

	handlerFactory := handlers.NewHandlerFactory(storeFactory, localizer, &upgrader)

	validators.Init()

	srv.RegisterHandler("/ws", handlerFactory.WsHandler().EntryPoint)

	srv.RegisterHandler("/languages", handlerFactory.LanguageHandler().All)
	srv.RegisterHandler("/activities", handlerFactory.ActivityHandler().All)

	srv.RegisterHandler("/conversations/message/send", handlerFactory.ConversationHandler().HandleMessage)
	srv.RegisterHandler("/conversations/delete", handlerFactory.UserHandler().IsLogged(
		handlerFactory.ConversationHandler().Delete),
	)

	srv.RegisterHandler("/pages", handlerFactory.PageHandler().All)
	srv.RegisterHandler("/pages/new", handlerFactory.UserHandler().IsLogged(
		handlerFactory.PageHandler().New),
	)
	srv.RegisterHandler("/pages/update", handlerFactory.UserHandler().IsLogged(
		handlerFactory.PageHandler().Update),
	)
	srv.RegisterHandler("/pages/publish", handlerFactory.UserHandler().IsLogged(
		handlerFactory.PageHandler().Publish),
	)
	srv.RegisterHandler("/pages/delete", handlerFactory.UserHandler().IsLogged(
		handlerFactory.PageHandler().Delete),
	)

	srv.RegisterHandler("/images/delete", handlerFactory.UserHandler().IsLogged(
		handlerFactory.ImageHandler().Delete),
	)

	// srv.RegisterHandler("/users", handlerFactory.UserHandler().All)

	srv.RegisterHandler("/profiles/update", handlerFactory.UserHandler().IsLogged(
		handlerFactory.ProfileHandler().Update),
	)
	srv.RegisterHandler("/profiles/mine", handlerFactory.UserHandler().IsLogged(
		handlerFactory.UserHandler().Profile),
	)
	srv.RegisterHandler("/profiles/pages", handlerFactory.UserHandler().IsLogged(
		handlerFactory.PageHandler().ProfilePages),
	)
	srv.RegisterHandler("/profile/conversations", handlerFactory.UserHandler().IsLogged(
		handlerFactory.ConversationHandler().ProfileConversations),
	)

	srv.RegisterHandler("/login", handlerFactory.UserHandler().Login)
	srv.RegisterHandler("/signup", handlerFactory.UserHandler().SignUp)
	srv.RegisterHandler("/logout", handlerFactory.UserHandler().IsLogged(
		handlerFactory.UserHandler().Logout),
	)

	srv.ServePublic(c.PublicPath)

	signalChan := make(chan os.Signal, 1)
	signalDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		<-signalChan
		log.Info("received os.Interrupt signal, stopping services")
		storeFactory.WsStore().Close(signalDone)
		<-signalDone
		if err := srv.HTTPServer.Shutdown(nil); err != nil {
			log.Panic(err)
		}
		log.Info("HTTPServer gracefully closed")

		close(signalDone)
	}()

	srv.Start()

	<-signalDone
}
