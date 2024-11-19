package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/xuri/excelize/v2"
)

// Question represents the structure of a quiz question
type Question struct {
	QID       string // Unique identifier for the question
	QChapter  string // The chapter this question belongs to
	QAnswer   string // The correct answer to the question
	QHirakata string // The question in Hirakata format (displayed to the user)
	QRomaji   string // The Romaji (phonetic representation) of the question
	QType     string // Type or category of the question
}

// loadQuestionsFromExcel reads the questions from an excel file
func loadQuestionsFromExcel(filepath string) ([]Question, error) {
	f, err := excelize.OpenFile(filepath) // Open the excel file
	if err != nil {
		return nil, err //Return error if file cannot be opened
	}

	var questions []Question
	rows, err := f.GetRows("Sheet1") //Read rows from the "Sheet1" worksheet
	if err != nil {
		return nil, err //return error if rows can be read
	}

	//Iterate over the rows starting from the second row (skip header)
	for _, row := range rows[1:] {
		if len(row) < 6 {
			continue //skip rows with incomplete data
		}
		question := Question{
			QID:       row[0],
			QChapter:  row[1],
			QAnswer:   row[2],
			QHirakata: row[3],
			QRomaji:   row[4],
			QType:     row[5],
		}
		questions = append(questions, question) //add questions to the list
	}

	return questions, nil
}

// getQuestionsByChapter filters questions based on the selected chapter
func getQuestionsByChapter(questions []Question, chapter string) []Question {
	var filtered []Question
	for _, q := range questions {
		if q.QChapter == chapter {
			filtered = append(filtered, q) // Add questions matching the chapter
		}
	}
	return filtered
}

// getRandomAnswers generates a list of random answers excluding the correct one
func getRandomAnswers(questions []Question, correctAnswer string, count int) []string {
	var answers []string
	for _, q := range questions {
		if q.QAnswer != correctAnswer {
			answers = append(answers, q.QAnswer) // Add wrong answers to the list
		}
	}

	//Shuffle the answers to randomize the selection
	rand.Shuffle(len(answers), func(i, j int) {
		answers[i], answers[j] = answers[j], answers[i]
	})

	//return only the requested number of answers
	return answers[:count]
}

func main() {
	rand.Seed(time.Now().UnixNano()) //Seed the random number generator

	//Load questions from the Excel file
	questions, err := loadQuestionsFromExcel("quizsheet.xlsx")
	if err != nil {
		log.Fatalf("Failed to load quiz questions: %v", err) //Exit if loading fails
	}

	//Initialize Fyne app
	a := app.New()
	w := a.NewWindow("Genki Quiz")
	w.Resize(fyne.NewSize(500, 400)) //set the window size

	//UI state variables
	score := 0            // Tracks the user's score
	currentChapter := "1" // Default chapter to "1"
	chapterSelector := widget.NewRadioGroup([]string{"1", "2", "3", "4"}, func(selected string) {
		currentChapter = selected // Update the chapter when user selects it
	})
	chapterSelector.SetSelected("1") //Set the default selected chapter

	//Create labels and containers for the UI
	questionLabel := widget.NewLabel("")                           // Displays the question
	romajiLabel := widget.NewLabel("")                             // Displays the Romaji (phonetic representation)
	optionsContainer := container.NewVBox()                        // Contains the answer buttons
	scoreLabel := widget.NewLabel(fmt.Sprintf("Score: %d", score)) // Displays the current score

	//Declare loadQuestion as a closure so it can access the UI elements
	var loadQuestion func()
	loadQuestion = func() {
		//Filter questions for the selected chapter
		chapterQuestions := getQuestionsByChapter(questions, currentChapter)
		if len(chapterQuestions) == 0 {
			//No questions available for this chapter
			questionLabel.SetText("No questions available for this chapter.")
			optionsContainer.Objects = nil //clear options
			optionsContainer.Refresh()
			return
		}

		//Select a random question from the filtered list
		q := chapterQuestions[rand.Intn(len(chapterQuestions))]
		questionLabel.SetText(q.QHirakata)                        //Set the question text
		romajiLabel.SetText(fmt.Sprintf("Romaji: %s", q.QRomaji)) //Display romaji

		// Generate 4 answer options (1 correct + 3 random wrong answers)
		randomAnswers := getRandomAnswers(chapterQuestions, q.QAnswer, 3)
		allAnswers := append(randomAnswers, q.QAnswer) // combine wrong and correct answer
		rand.Shuffle(len(allAnswers), func(i, j int) { //shuffle the answers
			allAnswers[i], allAnswers[j] = allAnswers[j], allAnswers[i]
		})

		//Clear and reload the options
		optionsContainer.Objects = nil
		for _, opt := range allAnswers {
			opt := opt //capture the loop variable
			button := widget.NewButton(opt, func() {
				//check if the selected answer is correct
				if opt == q.QAnswer {
					score++ //Increment score for correct answer
				}
				scoreLabel.SetText(fmt.Sprintf("Score: %d", score)) //Update score display
				loadQuestion()                                      //load the next question
			})
			optionsContainer.Add(button) //Add button to the container
		}
		optionsContainer.Refresh() // Refresh the container to display images
	}

	//Initial question load
	loadQuestion()

	//Set up the layout
	w.SetContent(container.NewVBox(
		widget.NewLabel("Select Chapter:"),
		chapterSelector,
		questionLabel,
		romajiLabel,
		optionsContainer,
		scoreLabel,
		widget.NewButton("Next Question", loadQuestion),
	))

	w.ShowAndRun()
}
