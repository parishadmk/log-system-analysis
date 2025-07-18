package main

import (
	"context"
	"encoding/json"
	"net/http"

	authpb "github.com/parishadmk/log-system-analysis/internal/api/auth"
	"go.uber.org/zap"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func runHTTPLogin(addr string, grpcClient authpb.AuthServiceClient, logger *zap.Logger) {
	http.HandleFunc("/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Only POST", http.StatusMethodNotAllowed)
			return
		}
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		ctx := context.Background()
		resp, err := grpcClient.Login(ctx, &authpb.LoginRequest{
			Username: req.Username,
			Password: req.Password,
		})
		if err != nil {
			logger.Warn("login failed", zap.Error(err))
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": resp.Token})
	})

	logger.Info("HTTP login endpoint listening", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Fatal("http server failed", zap.Error(err))
	}
}
