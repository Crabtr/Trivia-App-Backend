package main

import (
	"database/sql"
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
	question_body        VARCHAR(255) NOT NULL UNIQUE,
	category			 VARCHAR(255) NOT NULL,
	difficulty           TEXT NOT NULL,
	type                 TEXT NOT NULL,
	correct_answer       VARCHAR(255) NOT NULL,
	incorrect_answer_1	 VARCHAR(255) NOT NULL,
	incorrect_answer_2	 VARCHAR(255),
	incorrect_answer_3	 VARCHAR(255)
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

	for questionCount < 3369 {

		url := fmt.Sprintf("https://opentdb.com/api.php?amount=50")

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
			if myQuestions.Results[i].QuestionType == "boolean" {

				_, err = db.Exec(
					`INSERT INTO questions VALUES (?,?,?,?,?,?,?,?);`,
					myQuestions.Results[i].QuestionBody,
					myQuestions.Results[i].Category,
					myQuestions.Results[i].Difficulty,
					myQuestions.Results[i].QuestionType,
					myQuestions.Results[i].CorrectAnswer,
					myQuestions.Results[i].IncorrectAnswers[0],
					"NULL",
					"NULL",
				)
				if err != nil {
					fmt.Println(err)
					fmt.Println(myQuestions.Results[i])
				}

			} else if myQuestions.Results[i].QuestionType == "multiple" {

				_, err = db.Exec(
					`INSERT INTO questions VALUES (?,?,?,?,?,?,?,?);`,
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

		}
		err = db.QueryRow(allQuestionsCheck).Scan(&questionCount)
		if err != nil {
			panic(err)
		}
		if questionCount == 3369 {
			fmt.Println("LOADING TO DATABASE COMPLETE!!")
		}

	}

}
