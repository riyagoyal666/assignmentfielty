package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Student struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

var students = make(map[int]Student)

func createStudent(w http.ResponseWriter, r *http.Request) {
	var student Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}
	students[student.ID] = student
	json.NewEncoder(w).Encode(student)
}

func getAllStudents(w http.ResponseWriter, r *http.Request) {
	var allStudents []Student
	for _, student := range students {
		allStudents = append(allStudents, student)
	}
	json.NewEncoder(w).Encode(allStudents)
}

func getStudentByID(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if student, found := students[id]; found {
		json.NewEncoder(w).Encode(student)
	} else {
		http.Error(w, "Student not found", http.StatusNotFound)
	}
}

func updateStudentByID(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if _, found := students[id]; !found {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}
	var updatedStudent Student
	json.NewDecoder(r.Body).Decode(&updatedStudent)
	updatedStudent.ID = id
	students[id] = updatedStudent
	json.NewEncoder(w).Encode(updatedStudent)
}

func deleteStudentByID(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if _, found := students[id]; found {
		delete(students, id)
		w.WriteHeader(http.StatusNoContent)
	} else {
		http.Error(w, "Student not found", http.StatusNotFound)
	}
}

func generateSummaryWithOllama(prompt string) (string, error) {

	ollamaAPIURL := "http://localhost:11434/v1/ask"

	requestBody := map[string]interface{}{
		"model":  "llama3",
		"prompt": prompt,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to encode request body: %v", err)
	}

	resp, err := http.Post(ollamaAPIURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to make request to Ollama: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned an error: %s", string(body))
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err == nil {
		if summary, exists := response["text"].(string); exists {
			return summary, nil
		}
	}

	return string(body), nil
}

func generateManualSummary(student Student) string {

	return fmt.Sprintf("Student ID: %d, Name: %s, Age: %d, Email: %s", student.ID, student.Name, student.Age, student.Email)
}

func getStudentSummary(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(mux.Vars(r)["id"])
	if student, found := students[id]; found {

		prompt := fmt.Sprintf("Summarize the following student: ID: %d, Name: %s, Age: %d, Email: %s", student.ID, student.Name, student.Age, student.Email)

		summary, _ := generateSummaryWithOllama(prompt)
		if summary == "" {

			summary = generateManualSummary(student)
		}

		json.NewEncoder(w).Encode(map[string]string{"summary": summary})
	} else {
		http.Error(w, "Student not found", http.StatusNotFound)
	}
}

func main() {

	r := mux.NewRouter()

	r.HandleFunc("/students", createStudent).Methods("POST")
	r.HandleFunc("/students", getAllStudents).Methods("GET")
	r.HandleFunc("/students/{id}", getStudentByID).Methods("GET")
	r.HandleFunc("/students/{id}", updateStudentByID).Methods("PUT")
	r.HandleFunc("/students/{id}", deleteStudentByID).Methods("DELETE")
	r.HandleFunc("/students/{id}/summary", getStudentSummary).Methods("GET")

	fmt.Println("Server is running on port 8088...")
	log.Fatal(http.ListenAndServe(":8088", r))
}
