package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Server struct {
	sessionToken string
	sessionMutex sync.RWMutex
	lastAuth     time.Time
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ItemResponse struct {
	Data string `json:"data"`
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) ensureAuthenticated() error {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	if s.sessionToken != "" && time.Since(s.lastAuth) < 30*time.Minute {
		return nil
	}

	cmd := exec.Command("op", "whoami")
	if err := cmd.Run(); err != nil {
		log.Printf("1Password not authenticated, attempting signin...")
		signinCmd := exec.Command("op", "signin")
		if signinErr := signinCmd.Run(); signinErr != nil {
			return fmt.Errorf("failed to signin to 1Password: %v", signinErr)
		}
		
		whoamiCmd := exec.Command("op", "whoami")
		if whoamiErr := whoamiCmd.Run(); whoamiErr != nil {
			return fmt.Errorf("signin completed but still not authenticated: %v", whoamiErr)
		}
	}

	s.lastAuth = time.Now()
	return nil
}

func (s *Server) executeOpCommand(args ...string) ([]byte, error) {
	if err := s.ensureAuthenticated(); err != nil {
		return nil, err
	}

	cmd := exec.Command("op", args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("op command failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to execute op command: %v", err)
	}

	return output, nil
}

func (s *Server) handleItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	itemName := r.URL.Query().Get("name")
	vault := r.URL.Query().Get("vault")
	field := r.URL.Query().Get("field")

	if itemName == "" {
		s.writeErrorResponse(w, "item name is required", http.StatusBadRequest)
		return
	}

	args := []string{"item", "get", itemName, "--reveal"}
	if vault != "" {
		args = append(args, "--vault", vault)
	}
	if field != "" {
		args = append(args, "--field", field)
	}

	output, err := s.executeOpCommand(args...)
	if err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.writeDataResponse(w, string(output))
}

func (s *Server) handleRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reference := r.URL.Query().Get("ref")
	if reference == "" {
		s.writeErrorResponse(w, "reference is required", http.StatusBadRequest)
		return
	}

	output, err := s.executeOpCommand("read", reference)
	if err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.writeDataResponse(w, string(output))
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vault := r.URL.Query().Get("vault")
	category := r.URL.Query().Get("category")

	args := []string{"item", "list", "--format=json"}
	if vault != "" {
		args = append(args, "--vault", vault)
	}
	if category != "" {
		args = append(args, "--categories", category)
	}

	output, err := s.executeOpCommand(args...)
	if err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(output)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.ensureAuthenticated(); err != nil {
		s.writeErrorResponse(w, "1Password not authenticated", http.StatusServiceUnavailable)
		return
	}

	s.writeDataResponse(w, "OK")
}

func (s *Server) handleWhoami(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	output, err := s.executeOpCommand("whoami")
	if err != nil {
		s.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.writeDataResponse(w, strings.TrimSpace(string(output)))
}

func (s *Server) writeErrorResponse(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

func (s *Server) writeDataResponse(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ItemResponse{Data: strings.TrimSpace(data)})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := NewServer()

	http.HandleFunc("/item", server.handleItem)
	http.HandleFunc("/read", server.handleRead)
	http.HandleFunc("/list", server.handleList)
	http.HandleFunc("/health", server.handleHealth)
	http.HandleFunc("/whoami", server.handleWhoami)

	log.Printf("Starting 1Password server on port %s", port)
	log.Printf("Available endpoints:")
	log.Printf("  GET /item?name=<item>&vault=<vault>&field=<field> - Get item details")
	log.Printf("  GET /read?ref=<reference> - Read secret reference")
	log.Printf("  GET /list?vault=<vault>&category=<category> - List items")
	log.Printf("  GET /health - Check authentication status")
	log.Printf("  GET /whoami - Get current user")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}