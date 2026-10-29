package grpcserver

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"vacancy_notifications_bot/internal/model"

	pb "vacancy_notifications_bot/proto"

	"google.golang.org/grpc"
)

var logPrefix = "[GRPC] "

func logMessage(level slog.Level, message string, args ...any) {
	slog.Log(context.Background(), level, logPrefix+message, args...)
}

type Server struct {
	pb.UnimplementedSubscriptionsServer
}

func (server *Server) PushVacancies(
	ctx context.Context,
	req *pb.SendVacancies,
) (*pb.ConfirmPush, error) {
	receivedVacancies := req.GetVacancies()
	vacancies := make([]model.Vacancy, 0, len(receivedVacancies))
	for _, vacancy := range receivedVacancies {
		vacancies = append(vacancies, model.Vacancy{
			ID:         vacancy.GetId(),
			Title:      vacancy.GetTitle(),
			Experience: vacancy.GetExperience(),
			URL:        vacancy.GetUrl(),
		})
	}

	go func() {
		if err := processVacancies(vacancies); err != nil {
			logMessage(slog.LevelError, "Failed to process vacancies", "error", err)
		}
	}()

	logMessage(slog.LevelInfo, fmt.Sprintf("Received %d vacancies", len(vacancies)))

	return &pb.ConfirmPush{Ack: true}, nil
}

func StartServer() error {
	grpcPort, exists := os.LookupEnv("BOT_GRPC_PORT")
	if !exists {
		grpcPort = "50051"
		logMessage(slog.LevelInfo, "BOT_GRPC_PORT environment variable not set, starting server on default port 50051...")
	} else {
		logMessage(slog.LevelInfo, "Starting server on port "+grpcPort)
	}

	listener, err := net.Listen("tcp", "0.0.0.0:"+grpcPort)
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	pb.RegisterSubscriptionsServer(server, &Server{})

	logMessage(slog.LevelInfo, "gRPC server listening on port "+grpcPort)

	return server.Serve(listener)
}

func processVacancies(vacancies []model.Vacancy) error {
	logMessage(slog.LevelInfo, "\"Processed\" vacancies")
	return nil
}
