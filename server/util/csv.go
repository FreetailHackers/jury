package util

import ( 	
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"server/database"
	"server/models"
	"strings"
	"strconv"
 	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// Read CSV file and return a slice of judge structs
func ParseJudgeCSV(content string, hasHeader bool) ([]*models.Judge, error) {
	r := csv.NewReader(strings.NewReader(content))

	// Empty CSV file
	if content == "" {
		return []*models.Judge{}, nil
	}

	// If the CSV file has a header, skip the first line
	if hasHeader {
		r.Read()
	}

	// Read the CSV file, looping through each record
	var judges []*models.Judge
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Make sure the record has 3 elements (name, email, notes)
		if len(record) != 3 {
			return nil, fmt.Errorf("record does not contain 3 elements: '%s'", strings.Join(record, ","))
		}

		// Add judge to slice
		judges = append(judges, models.NewJudge(record[0], record[1], record[2]))
	}

	return judges, nil
}

// Read CSV file and return a slice of project structs
func ParseProjectCsv(content string, hasHeader bool, db *mongo.Database) ([]*models.Project, error) {
	r := csv.NewReader(strings.NewReader(content))

	// Empty CSV file
	if content == "" {
		return []*models.Project{}, nil
	}

	// Get the options from the database
	options, err := database.GetOptions(db)
	if err != nil {
		return nil, err
	}

	// If the CSV file has a header, skip the first line
	if hasHeader {
		r.Read()
	}

	// Read the CSV file, looping through each record
	var projects []*models.Project
	var locality int64 = 0

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Make sure the record has at least 4 elements (name, description, URL, Locality)
		if len(record) < 4 {
			return nil, fmt.Errorf("record contains less than 3 elements: '%s'", strings.Join(record, ","))
		}

		// Get the challenge list
		challengeList := []string{}
		if len(record) > 6 && record[6] != "" {
			challengeList = strings.Split(record[6], ",")
		}
		for i := range challengeList {
			challengeList[i] = strings.TrimSpace(challengeList[i])
		}

		// Optional fields
		var tryLink string
		if len(record) > 3 && record[3] != "" {
			tryLink = record[3]
		}
		var videoLink string
		if len(record) > 4 && record[4] != "" {
			videoLink = record[4]
		}
		if len(record) > 5 && record[5] != "" {
			locality, err = strconv.ParseInt(record[5], 10, 64)
			if err != nil {
				return nil, err
			}
			Localities = append(Localities, locality)
		}

		if options.NextTableNum < locality {
			options.NextTableNum = locality
		}

		// Add judge to slice
		projects = append(projects, models.NewProject(record[0], options.NextTableNum, record[1], record[2], tryLink, videoLink, locality, challengeList))

		// Increment the table number
		options.NextTableNum++
	}

	// Update the options table number in the database
	err = database.UpdateNextTableNum(db, context.Background(), options.NextTableNum)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

// Generate a workable CSV for Jury based on the output CSV from Devpost
// Columns:
//  0. Project Title - title
//  1. Submission Url - url
//  2. Project Status - Draft or Submitted (ignore drafts)
//  3. Judging Status - ignore
//  4. Highest Step Completed - ignore
//  5. Project Created At - ignore
//  6. About The Project - description
//  7. "Try it out" Links" - try_link
//  8. Video Demo Link - video_link
//  9. Opt-In Prizes - challenge_list
//  10. Built With - ignore
//  11. Notes - ignore
//  12. Team Colleges/Universities - ignore
//  13. Additional Team Member Count - ignore
//  14. (and remiaining rows) Custom questions - custom_questions (ignore for now)
func ParseDevpostCSV(content string, db *mongo.Database) ([]*models.Project, error) {
	r := csv.NewReader(strings.NewReader(content))

	// Empty CSV file
	if content == "" {
		return []*models.Project{}, nil
	}

	// Skip the first line
	r.Read()

	// Get the options from the database
	options, err := database.GetOptions(db)
	if err != nil {
		return nil, err
	}

	// Read the CSV file, looping through each record
	var projects []*models.Project
	var locality int64 = 0
	var dupeLocal = make(map[int64]bool) 
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		// Make sure the record has 14 or more elements (see above)
		if len(record) < 13 {
			return nil, fmt.Errorf("record does not contain 14 or more elements (invalid devpost csv): '%s'", strings.Join(record, ","))
		}

		// If the project is a Draft, skip it
		if record[2] == "Draft" {
			continue
		}

		// Split challenge list into a slice and trim them
		challengeList := strings.Split(record[10], ",")
		if record[10] == "" {
			challengeList = []string{}
		}
		for i := range challengeList {
			challengeList[i] = strings.TrimSpace(challengeList[i])
		}

		// Locality Field
		if len(record) > 9 && record[9] != "" {
			locality, err = strconv.ParseInt(record[9], 10, 64)
			if err != nil {
				return nil, err
			}
			if !dupeLocal[locality] {
				Localities = append(Localities, locality)
				dupeLocal[locality] = true
			}
		}

		if options.NextTableNum < locality {
			options.NextTableNum = locality
		}


		// Add project to slice
		projects = append(projects, models.NewProject(
			record[0],
			options.NextTableNum,
			record[6],
			record[1],
			record[7],
			record[8],
			locality,
			challengeList,
		))

		// Increment the table number
		options.NextTableNum++
	}

	// Update the options table number in the database
	err = database.UpdateNextTableNum(db, context.Background(), options.NextTableNum)
	if err != nil {
		return nil, err
	}

	return projects, nil
}

func AddCsvData(name string, content []byte, ctx *gin.Context) {
 	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", name))
 	ctx.Header("Content-Type", "text/csv")
 	ctx.Data(http.StatusOK, "text/csv", content)
 }

 // Create a CSV file from a list of judges
 func CreateJudgeCSV(judges []*models.Judge) []byte {
 	csvBuffer := &bytes.Buffer{}

 	// Create a new CSV writer
 	w := csv.NewWriter(csvBuffer)

 	// Write the header
 	w.Write([]string{"Name", "Email", "Notes", "Code", "Active", "ReadWelcome", "Notes", "Alpha", "Beta", "LastActivity"})

 	// Write each judge
 	for _, judge := range judges {
 		w.Write([]string{judge.Name, judge.Email, judge.Notes, judge.Code, fmt.Sprintf("%t", judge.Active), fmt.Sprintf("%t", judge.ReadWelcome), judge.Notes, fmt.Sprintf("%f", judge.Alpha), fmt.Sprintf("%f", judge.Beta), fmt.Sprintf("%d", judge.LastActivity)})
 	}

 	// Flush the writer
 	w.Flush()

 	return csvBuffer.Bytes()
 }

 // Create a CSV file from a list of projects
 func CreateProjectCSV(projects []*models.Project) []byte {
 	csvBuffer := &bytes.Buffer{}

 	// Create a new CSV writer
 	w := csv.NewWriter(csvBuffer)

 	// Write the header
 	w.Write([]string{"Name", "Table", "Description", "URL", "TryLink", "VideoLink", "ChallengeList", "Mu", "SigmaSq", "Active", "LastActivity"})

 	// Write each project
 	for _, project := range projects {
 		w.Write([]string{project.Name, fmt.Sprintf("Table %d", project.Location), project.Description, project.Url, project.TryLink, project.VideoLink, strings.Join(project.ChallengeList, ","), fmt.Sprintf("%f", project.Mu), fmt.Sprintf("%f", project.SigmaSq), fmt.Sprintf("%t", project.Active), fmt.Sprintf("%d", project.LastActivity)})
 	}

 	// Flush the writer
 	w.Flush()

 	return csvBuffer.Bytes()
 }