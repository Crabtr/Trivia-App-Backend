package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

type results struct {
	ResponseCode int              `json:"response_code"`
	Results      []QuestionStruct `json:"results"`
}

type QuestionStruct struct {
	Category         string   `json:"category"`
	QuestionType     string   `json:"type"`
	Difficulty       string   `json:"difficulty"`
	QuestionBody     string   `json:"question"`
	CorrectAnswer    string   `json:"correct_answer"`
	IncorrectAnswers []string `json:"incorrect_answers"`
}

func main() {
	dbPath := fmt.Sprintf(
		"%s:%s@tcp(%s)/trivia?charset=utf8mb4",
		"root",     // username
		"rootpass", // password
		"0.0.0.0",  // address
	)

	db, err := sql.Open("mysql", dbPath)
	if err != nil {
		panic(err)
	}

	createQuestionsStmt := `
	CREATE TABLE IF NOT EXISTS questions
	(
	id					 INT          NOT NULL AUTO_INCREMENT,
	question_body        VARCHAR(255) NOT NULL UNIQUE,
	category			 VARCHAR(255) NOT NULL,
	difficulty           TEXT NOT NULL,
	type                 TEXT NOT NULL,
	correct_answer       VARCHAR(255) NOT NULL,
	incorrect_answer_1	 VARCHAR(255) NOT NULL,
	incorrect_answer_2	 VARCHAR(255),
	incorrect_answer_3	 VARCHAR(255),
	PRIMARY KEY (id)
	);`

	_, err = db.Exec(createQuestionsStmt)
	if err != nil {
		panic(err)
	}
	allQuestionsCheck := `
	SELECT COUNT(*)
	FROM questions;
	`

	questionCount := 0

	for questionCount <= 2909 {

		url := fmt.Sprintf("https://opentdb.com/api.php?amount=50&type=multiple&encode=base64")

		respone, err := http.Get(url)
		if err != nil {
			panic(err)
		}
		data, _ := ioutil.ReadAll(respone.Body)

		var myQuestions results

		err = json.Unmarshal([]byte(string(data)), &myQuestions)
		if err != nil {
			panic(err)
		}

		for i := 0; i <= len(myQuestions.Results)-1; i++ {

			data, err := base64.StdEncoding.DecodeString(myQuestions.Results[i].QuestionBody)
			myQuestions.Results[i].QuestionBody = string(data)

			if err != nil {

				panic(err)
			}

			data, err = base64.StdEncoding.DecodeString(myQuestions.Results[i].Category)
			myQuestions.Results[i].Category = string(data)

			if err != nil {

				panic(err)
			}
			data, err = base64.StdEncoding.DecodeString(myQuestions.Results[i].Difficulty)
			myQuestions.Results[i].Difficulty = string(data)

			if err != nil {

				panic(err)
			}
			data, err = base64.StdEncoding.DecodeString(myQuestions.Results[i].QuestionType)
			myQuestions.Results[i].QuestionType = string(data)

			if err != nil {

				panic(err)
			}
			data, err = base64.StdEncoding.DecodeString(myQuestions.Results[i].CorrectAnswer)
			myQuestions.Results[i].CorrectAnswer = string(data)

			if err != nil {

				panic(err)
			}
			data, err = base64.StdEncoding.DecodeString(myQuestions.Results[i].IncorrectAnswers[0])
			myQuestions.Results[i].IncorrectAnswers[0] = string(data)

			if err != nil {

				panic(err)
			}

			data, err = base64.StdEncoding.DecodeString(myQuestions.Results[i].IncorrectAnswers[1])
			myQuestions.Results[i].IncorrectAnswers[1] = string(data)

			if err != nil {

				panic(err)
			}

			data, err = base64.StdEncoding.DecodeString(myQuestions.Results[i].IncorrectAnswers[2])
			myQuestions.Results[i].IncorrectAnswers[2] = string(data)

			if err != nil {

				panic(err)
			}

			_, err = db.Exec(
				`INSERT INTO questions (question_body, category, difficulty, type, correct_answer, incorrect_answer_1,incorrect_answer_2,incorrect_answer_3) VALUES (?,?,?,?,?,?,?,?);`,
				myQuestions.Results[i].QuestionBody,
				myQuestions.Results[i].Category,
				myQuestions.Results[i].Difficulty,
				myQuestions.Results[i].QuestionType,
				myQuestions.Results[i].CorrectAnswer,
				myQuestions.Results[i].IncorrectAnswers[0],
				myQuestions.Results[i].IncorrectAnswers[1],
				myQuestions.Results[i].IncorrectAnswers[2],
			)
			if err != nil {
				fmt.Println(err)
				fmt.Println(myQuestions.Results[i])
			}

		}
		err = db.QueryRow(allQuestionsCheck).Scan(&questionCount)
		if err != nil {
			panic(err)
		}

	}

	fmt.Println("LOADING DATABASE COMPLETE")

}
