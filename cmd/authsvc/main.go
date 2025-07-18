package main

import (
    "context"
    "fmt"
    "net"
    "os"
    "time"

    "github.com/spf13/viper"
    "go.uber.org/zap"
    "google.golang.org/grpc"
    "golang.org/x/crypto/bcrypt"
    "github.com/golang-jwt/jwt/v4"

    "github.com/parishadmk/log-system-analysis/internal/api/auth"
    "github.com/parishadmk/log-system-analysis/internal/lib"
    "github.com/jackc/pgx/v5/pgxpool"
)

type server struct {
    auth.UnimplementedAuthServiceServer
    db     *pgxpool.Pool
    logger *zap.Logger
    jwtKey []byte
}

func initConfig() {
    viper.SetConfigFile("/etc/authsvc/config.yml")
    if err := lib.LoadConfig(viper.ConfigFileUsed()); err != nil {
       	panic(fmt.Sprintf("failed to load config: %v", err))
    }
}

func main() {
    // 1) Logger
    if err := lib.InitLogger(); err != nil {
        fmt.Fprintf(os.Stderr, "logger init: %v\n", err)
        os.Exit(1)
    }
    logger := lib.Log
    defer logger.Sync()

    // 2) Config
    initConfig()
    dsn := viper.GetString("cockroach.dsn")
    port := viper.GetString("server.port")
    jwtKey := []byte(viper.GetString("server.jwt_key"))
    if len(jwtKey) == 0 {
        logger.Fatal("server.jwt_key must be set in config")
    }

    // 3) Connect Cockroach
    pool, err := lib.NewCockroachPool(dsn)
    if err != nil {
        logger.Fatal("cockroach connect failed", zap.Error(err))
    }
    defer pool.Close()

    // 4) gRPC server
    lis, err := net.Listen("tcp", ":"+port)
    if err != nil {
        logger.Fatal("listen failed", zap.Error(err))
    }
    grpcServer := grpc.NewServer()
    auth.RegisterAuthServiceServer(grpcServer, &server{
        db:     pool,
        logger: logger,
        jwtKey: jwtKey,
    })

    logger.Info("Auth Service listening", zap.String("port", port))
    if err := grpcServer.Serve(lis); err != nil {
        logger.Fatal("gRPC serve failed", zap.Error(err))
    }
}

// Login checks username/password and returns a JWT
func (s *server) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
    var (
        userID string
        hash   string
    )
    err := s.db.QueryRow(ctx,
        `SELECT id, hashed_password FROM users WHERE username=$1`,
        req.Username,
    ).Scan(&userID, &hash)
    if err != nil {
        return nil, err
    }
    if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
        return nil, err
    }

    // issue JWT
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": userID,
        "exp":     time.Now().Add(72 * time.Hour).Unix(),
    })
    signed, err := token.SignedString(s.jwtKey)
    if err != nil {
        return nil, err
    }
    return &auth.LoginResponse{Token: signed}, nil
}

// ValidateApiKey checks project_id + api_key
func (s *server) ValidateApiKey(ctx context.Context, req *auth.ApiKeyRequest) (*auth.ApiKeyResponse, error) {
    var ok bool
    err := s.db.QueryRow(ctx,
        `SELECT EXISTS (SELECT 1 FROM projects WHERE id=$1 AND api_key=$2)`,
        req.ProjectId, req.ApiKey,
    ).Scan(&ok)
    if err != nil {
        return nil, err
    }
    return &auth.ApiKeyResponse{Valid: ok}, nil
}