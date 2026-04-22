package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"

	"tekplat-crud/internal/config"
	"tekplat-crud/internal/store"
	"tekplat-crud/internal/web"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()

	db, err := openDatabase(cfg.DatabaseDSN())
	if err != nil {
		log.Fatalf("gagal konek database: %v", err)
	}
	defer db.Close()

	userStore := store.NewUserStore(db)
	tableStore := store.NewTableStore(db)
	app := web.NewApp(cfg, userStore, tableStore)

	server := &http.Server{
		Addr:         cfg.AppAddr,
		Handler:      app.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("server berjalan di %s", cfg.AppAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("gagal menjalankan server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown tidak bersih: %v", err)
	}
}

func openDatabase(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
