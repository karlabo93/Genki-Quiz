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
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	// Load quiz questions from an Excel file
	questions, err := loadQuestionsFromExcel("quizsheet.xlsx")
	if err != nil {
		log.Fatalf("Failed to load quiz questions: %v", err)
	}

	// Initialize the Fyne application
	a := app.New()
	w := a.NewWindow("Genki Quiz")
	w.Resize(fyne.NewSize(500, 400)) // Set the window size

	// Variables to track the game's state
	score := 0                            // User's score
	currentChapter := ""                  // Currently selected chapter
	var questionContainer *fyne.Container // Container for dynamically switching views

	// UI elements for the quiz game
	questionLabel := widget.NewLabel("")                           // Label to display the current question
	romajiLabel := widget.NewLabel("")                             // Label to display the romaji
	optionsContainer := container.NewVBox()                        // Container for answer options
	scoreLabel := widget.NewLabel(fmt.Sprintf("Score: %d", score)) // Display user's score

	// Function placeholders for dynamic behavior
	var loadQuestion func()         // Function to load the next question
	var showChapterSelection func() // Function to show chapter selection menu

	// Layout for the main quiz game
	gameLayout := func() fyne.CanvasObject {
		return container.NewVBox(
			container.NewCenter(widget.NewLabel("Genki Quiz!")), //Center the title
			questionLabel,    // Display the question
			romajiLabel,      // Display the romaji (phoenetic hint)
			optionsContainer, // Buttons for answer choices
			scoreLabel,       // Display the score
			widget.NewButton("Change Chapter", func() {
				// Show chapter selection menu when the button is clicked
				showChapterSelection()
			}),
			widget.NewButton("Next Question", loadQuestion),
		)
	}

	// Chapter selection menu
	showChapterSelection = func() {
		// Clear the current view and show chapter selection options
		questionContainer.Objects = []fyne.CanvasObject{
			container.NewCenter(container.NewVBox(
				widget.NewLabel("Select Chapter:"), // Prompt to select a chapter
				widget.NewRadioGroup([]string{"1", "2", "3", "4"}, func(selected string) {
					//Update the selected chapter and switch to quiz view
					currentChapter = selected
					questionContainer.Objects = []fyne.CanvasObject{gameLayout()} // Load the game layout
					questionContainer.Refresh()                                   // Refresh to apply changes
					loadQuestion()
				}),
			)),
		}
		questionContainer.Refresh() // Refresh the container to show the menu

	}

	// Load a new question based on the selected chapter
	loadQuestion = func() {
		// Filter questions by the selected chapter
		chapterQuestions := getQuestionsByChapter(questions, currentChapter)
		if len(chapterQuestions) == 0 {
			//Display a message if no questions are available
			questionLabel.SetText("No questions available for this chapter.")
			optionsContainer.Objects = nil
			optionsContainer.Refresh()
			return
		}

		//Randomly pick a question
		q := chapterQuestions[rand.Intn(len(chapterQuestions))]
		questionLabel.SetText(q.QHirakata)                        // Set the question text
		romajiLabel.SetText(fmt.Sprintf("Romaji: %s", q.QRomaji)) // Display the romaji

		// Generate 4 answer options (1 correct + 3 random wrong answers)
		randomAnswers := getRandomAnswers(chapterQuestions, q.QAnswer, 3)
		allAnswers := append(randomAnswers, q.QAnswer) // Combine correct and wrong answers
		rand.Shuffle(len(allAnswers), func(i, j int) { // Shuffle the options
			allAnswers[i], allAnswers[j] = allAnswers[j], allAnswers[i]
		})

		// Clear and populate the options container
		optionsContainer.Objects = nil
		for _, opt := range allAnswers {
			opt := opt // Capture the loop variable
			button := widget.NewButton(opt, func() {
				//Check if the selected answer is correct
				if opt == q.QAnswer {
					score++ // Increment score for a correct answer
				}
				scoreLabel.SetText(fmt.Sprintf("Score: %d", score)) // Update the score display
				loadQuestion()                                      // Load the next question
			})
			optionsContainer.Add(button) // Add the button to the container
		}
		optionsContainer.Refresh() // Refresh the container to display the buttons
	}
	questionContainer = container.NewVBox() // Create a container for dynamic content
	showChapterSelection()                  // Show the chapter selection menu initially
	w.SetContent(questionContainer)         // Set the window content

	// Start the application
	w.ShowAndRun()
}
