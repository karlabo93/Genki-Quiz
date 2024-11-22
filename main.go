package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/xuri/excelize/v2"
)

// Question represents a single quiz question with all its attributes
type Question struct {
	QID       string // Unique identifier for the question
	QChapter  string // Chapter number the question belongs to
	QAnswer   string // Correct answer
	QHirakata string // Question text in Japanese characters
	QRomaji   string // Question text in romanized form
	QType     string // Category or type of question
}

// gameState tracks the current state of the quiz
type gameState struct {
	score            int        // Current score
	questionsAsked   int        // Number of questions completed
	totalQuestions   int        // Total questions in current quiz
	currentChapter   string     // Selected chapter
	chapterQuestions []Question // Questions filtered for current chapter
}

// loadQuestionsFromExcel reads and parses questions from an Excel file
func loadQuestionsFromExcel(filepath string) ([]Question, error) {
	f, err := excelize.OpenFile(filepath)
	if err != nil {
		return nil, err
	}

	var questions []Question
	rows, err := f.GetRows("Sheet1")
	if err != nil {
		return nil, err
	}

	// Skip header row and process each data row
	for _, row := range rows[1:] {
		if len(row) < 6 {
			continue // Skip incomplete rows
		}
		question := Question{
			QID:       row[0],
			QChapter:  row[1],
			QAnswer:   row[2],
			QHirakata: row[3],
			QRomaji:   row[4],
			QType:     row[5],
		}
		questions = append(questions, question)
	}

	return questions, nil
}

// getQuestionsByChapter filters questions for a specific chapter
func getQuestionsByChapter(questions []Question, chapter string) []Question {
	var filtered []Question
	for _, q := range questions {
		if q.QChapter == chapter {
			filtered = append(filtered, q)
		}
	}
	return filtered
}

// getRandomAnswers generates wrong answer options, avoiding duplicates
func getRandomAnswers(questions []Question, correctAnswer string, count int) []string {
	var answers []string
	usedAnswers := make(map[string]bool)
	usedAnswers[correctAnswer] = true

	// Collect unique wrong answers
	for _, q := range questions {
		if !usedAnswers[q.QAnswer] {
			answers = append(answers, q.QAnswer)
			usedAnswers[q.QAnswer] = true
		}
	}

	// Randomize answers
	rand.Shuffle(len(answers), func(i, j int) {
		answers[i], answers[j] = answers[j], answers[i]
	})

	// Return requested number of wrong answers
	if len(answers) > count {
		return answers[:count]
	}
	return answers
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Load questions from Excel file
	questions, err := loadQuestionsFromExcel("quizsheet.xlsx")
	if err != nil {
		log.Fatalf("Failed to load quiz questions: %v", err)
	}

	// Initialize Fyne application and window
	a := app.New()
	w := a.NewWindow("Genki Quiz")
	w.Resize(fyne.NewSize(500, 400))

	// Initialize game state and UI elements
	state := &gameState{}
	var questionContainer *fyne.Container
	questionLabel := canvas.NewText("", theme.TextColor())
	questionLabel.TextStyle = fyne.TextStyle{Bold: true}
	questionLabel.TextSize = 24
	romajiLabel := widget.NewLabel("")
	optionsContainer := container.NewVBox()
	scoreLabel := widget.NewLabel("")
	var clickableRomajiLabel *widget.Button

	romajiLabel.Hide() // Initially hide romaji text

	// Forward declarations for UI navigation functions
	var loadQuestion func()
	var showChapterSelection func()
	var showQuizTypeSelection func()
	var showQuizSummary func()

	// Creates main quiz game layout
	gameLayout := func() fyne.CanvasObject {
		romajiVisible := false

		// Toggle button for showing/hiding romaji
		clickableRomajiLabel = widget.NewButton("Show Romaji", func() {
			romajiVisible = !romajiVisible
			if romajiVisible {
				romajiLabel.Show()
				clickableRomajiLabel.SetText("Hide Romaji")
			} else {
				romajiLabel.Hide()
				clickableRomajiLabel.SetText("Show Romaji")
			}
			romajiLabel.Refresh()
			clickableRomajiLabel.Refresh()
		})
		clickableRomajiLabel.Importance = widget.LowImportance

		// Progress and score tracking
		progressLabel := widget.NewLabel(fmt.Sprintf("Question %d/%d", state.questionsAsked+1, state.totalQuestions))
		scoreLabel.SetText(fmt.Sprintf("Score: %d/%d", state.score, state.questionsAsked))

		// Arrange UI elements vertically
		return container.NewVBox(
			container.NewCenter(widget.NewLabelWithStyle(
				fmt.Sprintf("Genki Quiz! (Chapter: %s)", state.currentChapter),
				fyne.TextAlignCenter,
				fyne.TextStyle{Bold: true},
			)),
			container.NewCenter(progressLabel),
			container.NewCenter(questionLabel),
			container.NewCenter(romajiLabel),
			container.NewCenter(clickableRomajiLabel),
			optionsContainer,
			scoreLabel,
		)
	}

	// Shows quiz completion screen with final score
	showQuizSummary = func() {
		questionContainer.Objects = []fyne.CanvasObject{
			container.NewCenter(container.NewVBox(
				widget.NewLabelWithStyle(
					"Quiz Complete!",
					fyne.TextAlignCenter,
					fyne.TextStyle{Bold: true},
				),
				widget.NewLabel(fmt.Sprintf("Final Score: %d/%d (%.1f%%)",
					state.score,
					state.totalQuestions,
					float64(state.score)/float64(state.totalQuestions)*100,
				)),
				widget.NewButton("Return to Chapter Selection", func() {
					state.score = 0
					state.questionsAsked = 0
					showChapterSelection()
				}),
			)),
		}
		questionContainer.Refresh()
	}

	// Shows quiz type selection screen (mini or full chapter)
	showQuizTypeSelection = func() {
		state.chapterQuestions = getQuestionsByChapter(questions, state.currentChapter)

		questionContainer.Objects = []fyne.CanvasObject{
			container.NewCenter(container.NewVBox(
				widget.NewLabel(fmt.Sprintf("Chapter %s - Available Questions: %d",
					state.currentChapter, len(state.chapterQuestions))),
				widget.NewButton("Mini Quiz (10 questions)", func() {
					state.totalQuestions = 10
					if len(state.chapterQuestions) < 10 {
						state.totalQuestions = len(state.chapterQuestions)
					}
					questionContainer.Objects = []fyne.CanvasObject{gameLayout()}
					questionContainer.Refresh()
					loadQuestion()
				}),
				widget.NewButton("Full Chapter Quiz", func() {
					state.totalQuestions = len(state.chapterQuestions)
					questionContainer.Objects = []fyne.CanvasObject{gameLayout()}
					questionContainer.Refresh()
					loadQuestion()
				}),
				widget.NewButton("Back to Chapter Selection", func() {
					showChapterSelection()
				}),
			)),
		}
		questionContainer.Refresh()
	}

	// Shows chapter selection screen
	showChapterSelection = func() {
		questionContainer.Objects = []fyne.CanvasObject{
			container.NewCenter(container.NewVBox(
				widget.NewLabelWithStyle(
					"Welcome to Genki Quiz!",
					fyne.TextAlignCenter,
					fyne.TextStyle{Bold: true},
				),
				widget.NewLabel("Select Chapter:"),
				widget.NewRadioGroup([]string{"1", "2", "3", "4"}, func(selected string) {
					state.currentChapter = selected
					showQuizTypeSelection()
				}),
			)),
		}
		questionContainer.Refresh()
	}

	// Loads and displays a new question
	loadQuestion = func() {
		if state.questionsAsked >= state.totalQuestions {
			showQuizSummary()
			return
		}

		// Create question pool
		availableQuestions := make([]Question, len(state.chapterQuestions))
		copy(availableQuestions, state.chapterQuestions)

		// Select random question and set up display
		q := availableQuestions[rand.Intn(len(availableQuestions))]
		questionLabel.Text = q.QHirakata
		questionLabel.Refresh()
		romajiLabel.SetText(q.QRomaji)

		// Generate and shuffle answer options
		randomAnswers := getRandomAnswers(availableQuestions, q.QAnswer, 3)
		allAnswers := append(randomAnswers, q.QAnswer)
		rand.Shuffle(len(allAnswers), func(i, j int) {
			allAnswers[i], allAnswers[j] = allAnswers[j], allAnswers[i]
		})

		// Create answer buttons
		optionsContainer.Objects = nil
		var correctButton *widget.Button

		for _, opt := range allAnswers {
			opt := opt
			var button *widget.Button
			button = widget.NewButton(opt, func() {
				state.questionsAsked++
				if opt == q.QAnswer {
					state.score++
					button.SetText(fmt.Sprintf("✅ %s", button.Text))
				} else {
					button.SetText(fmt.Sprintf("❌ %s", button.Text))
				}
				button.Refresh()

				// Show correct answer if wrong choice selected
				if correctButton != nil && correctButton != button {
					correctButton.SetText(fmt.Sprintf("✅ %s", correctButton.Text))
					correctButton.Refresh()
				}

				// Update score display
				scoreLabel.SetText(fmt.Sprintf("Score: %d/%d", state.score, state.questionsAsked))

				// Disable all buttons after answer
				for _, obj := range optionsContainer.Objects {
					if btn, ok := obj.(*widget.Button); ok {
						btn.OnTapped = nil
					}
				}

				// Load next question after delay
				time.AfterFunc(2*time.Second, loadQuestion)
			})

			if opt == q.QAnswer {
				correctButton = button
			}

			optionsContainer.Add(button)
		}
		optionsContainer.Refresh()
	}

	// Initialize and start application
	questionContainer = container.NewVBox()
	showChapterSelection()
	w.SetContent(questionContainer)
	w.ShowAndRun()
}
