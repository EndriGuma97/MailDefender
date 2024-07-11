// 100 a day ideally
//
//export email results sending or not see github for inspiration
package main

import (
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
)

var ListInvalidEmails []string
var clearList []string

var verifier = emailverifier.NewVerifier()
func isValidEmail(email string) (bool, error) {
	ret, err := verifier.Verify(email)
	if err != nil {
		fmt.Printf("Error verifying email: %s\n", err)
		return false, err
	}
	return ret.Syntax.Valid, nil
}
func handler(w http.ResponseWriter, r *http.Request) {
	var EmailErr error
	var ShareLink bool
    // Parse the form
    err := r.ParseForm()
    if err != nil {
        http.Error(w, "Error parsing form", http.StatusBadRequest)
        return
    }

    if r.Method == "POST" {
		ListInvalidEmails = clearList
		ShareLink = true
        file, _, err := r.FormFile("csvFile")
        if err != nil {
            http.Error(w, "Error retrieving file", http.StatusBadRequest)
            return
        }
        defer file.Close()

        // Read and parse the CSV file
        reader := csv.NewReader(file)
        for {
            record, err := reader.Read()
            if err == io.EOF {
                break
            }
            if err != nil {
                http.Error(w, "Error reading CSV file", http.StatusInternalServerError)
                return
            }

            if len(record) == 0 || strings.TrimSpace(record[0]) == "" {
                // Skip empty lines or records with empty first column
                continue
            }

            sendEmailTo := strings.TrimSpace(record[0]) // Assuming the email is in the first column
			firstName := strings.TrimSpace(record[1])
			lastName := strings.TrimSpace(record[2])

			isValid, errMail := isValidEmail(sendEmailTo)
			if !isValid {
				fmt.Printf("Invalid email: %s\n", sendEmailTo)
				var v string
				if errMail != nil {
					v = fmt.Sprintf("%v", errMail)
				} else {
					v = "Invalid Email"
				}
				
				ListInvalidEmails = append(ListInvalidEmails, sendEmailTo + `: ` + v)
				continue
			} else {
				ListInvalidEmails = append(ListInvalidEmails, sendEmailTo + " sent succesfully")
			}
            // Log the email to verify it's being read correctly
			
            //sendPersonalizedEmail("smtp.hostinger.com", "465", "info@bcbrillantemarketing.com", "Assembly3637997Ab,", "Test 1", "<div>Hello world!</div>", sendEmailTo)
           EmailErr = sendPersonalizedEmail(r.Form["smtpHost"][0], r.Form["smtpPort"][0], r.Form["email"][0], r.Form["password"][0], r.Form["subject"][0], r.Form["body"][0], sendEmailTo, firstName, lastName)
		   time.Sleep(15 * time.Second)
        }
    }

    // Parse and execute the template
    tpl := template.Must(template.ParseFiles("index.html"))
	//EmailErr (var EmailErr error)
	//ListInvalidEmails (var ListInvalidEmails []string)
		
    err = tpl.ExecuteTemplate(w, "index.html", struct {
		EmailErr          error
		ListInvalidEmails []string
		ShareLink bool

	}{
		EmailErr:          EmailErr,
		ListInvalidEmails: ListInvalidEmails,
		ShareLink: ShareLink,
	})
    if err != nil {
        http.Error(w, "Error executing template", http.StatusInternalServerError)
    }
}
func init() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/download", flushList)
	http.HandleFunc("/clear", clearListFunc)
    http.ListenAndServe(":8000", nil)
}
func main() {

}
func sendPersonalizedEmail(SmtpHost string, SmtpPort string, Email string, Password string, Subject string, Body string, To string, FirstName string, LastName string ) error {
	smtpHost := SmtpHost
	smtpPort := SmtpPort
	email := Email
	password := Password

	subject := Subject
	/*
	if len(FirstName) > 1 {
		subject = Subject + ", " + FirstName
	} */
	parts := strings.Split(Body, "</body>")
	if len(parts) < 2 {
        return fmt.Errorf("error: HTML content is not well-formed")
    }

	unsubForm := "forms.zohopublic.com/amskuportal/form/UnsubscribefromOurMailingList/formperma/VtavrbifIdOVbUFr8qNkkAshi1rwP3S6ykM2tzXZDAg?email=" + To
    part1 := parts[0]
    part2 := "</body>" + parts[1] 
	body := part1 + `<div><img src="https://script.google.com/macros/s/AKfycby5aW5uWDqb2C52gUU5rIYMpLSZtYLj3GGC6y5U0ZUHeigvAnRyoyA5_3-Tymf4u1cjVQ/exec?id=`+To+`&emailSubject=`+Subject+`" width="1" height="1" style="display: none; border: 0px; outline: none; text-decoration: none;"> </div>` + part2
	body = strings.ReplaceAll(body, "www.unsub.com", unsubForm)
	body = strings.ReplaceAll(body, "${FULLNAME}", FirstName + " " + LastName + ",")
	//body = strings.ReplaceAll(body, "webinar.getmskultrasound.com", "webinar.getmskultrasound.com/" + FirstName + "-" + LastName)

	headerMap := map[string]string{
		"From":         "Colin Rigney-AMSKU <" + email + ">",
		"To":           To,
		"Subject":      subject,
		"MIME-version": "1.0;",
		"Content-Type": "text/html; charset=\"UTF-8\";",
	}
	message := ""
	for k, v := range headerMap {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	auth := smtp.PlainAuth("", email, password, smtpHost)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // In production, use proper certificate verification.
		ServerName:         smtpHost,
	}

	conn, err := tls.Dial("tcp", smtpHost+":"+smtpPort, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %v", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}

	if err = client.Mail(email); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}
	if err = client.Rcpt(To); err != nil {
		return fmt.Errorf("failed to set recipient: %v", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to send data command: %v", err)
	}
	_, err = writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message: %v", err)
	}
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	err = client.Quit()
	if err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	fmt.Println("Personalized email sent successfully to", To)
	return nil
}

func flushList(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=emailList.csv")
	writer := csv.NewWriter(w)
	for _, v := range ListInvalidEmails {
		record := []string{v}
		writer.Write(record)
	}
	writer.Flush()
}

func clearListFunc(w http.ResponseWriter, _*http.Request) {
	ListInvalidEmails = clearList
}