package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
)

type interactionsWithJSONFile interface {
	readFile(fileName *os.File, jsonFile *JSONfiles) (JSONfiles, error)
	rewriteFile(fileName *os.File, pos int, resultText string) error
	extractCommandFromParams(posName, posParam int) (string, string)
	createFile(posName, posParam int) string
	renameFile(posName, posParam int) string
	deleteFile(posName, posParam int) string
	getFileCreationTime(posName, posParam int) string
	writeTextInFile(posName, posParam int) string
	conditions(posName, posParam int) (string, string)
	goToNextActionOrCondition(posName, posParam int, nextAction *string) string
}

type JSONfiles struct {
	JSONfiles []JSONfile `json:"JSONfiles"`
}

type JSONfile struct {
	Name              string   `json:"name"`
	ResultOfExectuion string   `json:"resultOfExectuion"`
	Params            []string `json:"params"`
}

const initFileName = "Actions.json"

//Actions.json - основной файл, куда записываются результаты выполнения тех или иных команд, а также где меняются условия
//ExampleJSONFile.json - файл, где можно в удобном формате задать изначальные параметра JSON файла
//Примеры всех возможных запросов указаны в ExampleJSONFile.json
//Прочитать результат выполнение операций каждого действия/условия можно в поле JSON "resultOfExectuion"
//У неочевиднх операций есть комментарии

var Commands = []string{"Create", "Rename", "Delete", "Get", "Write", "If", "Go"}

//Запись команды следует начинать с одного из этих слов, каждое слово соответсвует Определённой операции
//Create: Создать файл; Rename: Переименовать файл; Delete: Удалить файл; Get: Получить время создание файла;
//Write: Записать произвольную строку в файл; If: выполнение условия; Go: перейти к следующему условию/действию
//Также перед названием файла нужно писать слово file а перед вводимой строкой слово text

func (jsonFile *JSONfiles) readFile(fileName *os.File) error {
	buf, err := ioutil.ReadAll(fileName)
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, &jsonFile)
	return err
}

// Метод rewriteFile нужна, чтобы можно было занести результаты выполнения операции в структуру JSONFiles в поле "resultOfExectuion"
// Если действие не может быть выполнено, то в результатах действия указно пояснение и сама ошибка, при этом программа не прерывается
func (jsonFile *JSONfiles) rewriteFile(posName int, resultText string) error {
	jsonFile.JSONfiles[posName].ResultOfExectuion = resultText
	content, err := json.Marshal(jsonFile)
	if err != nil {
		return fmt.Errorf("Unable to unmarshal while rewrite JSON: %e", err)
	}
	err = ioutil.WriteFile(initFileName, content, 0644)
	if err != nil {
		return fmt.Errorf("Unable to rewrite JSON: %e", err)
	}
	return nil
}

// метод extractCommandFromParams нужна, чтобы можно было изначально понимать с какой коммандой идёт взаимодействие
func (jsonFile *JSONfiles) extractCommandFromParams(posName, posParam int) (string, string) {
	commandString := jsonFile.JSONfiles[posName].Params[posParam]
	for _, el := range Commands {
		if strings.HasPrefix(commandString, el) {
			return el, ""
		}
	}
	return "", fmt.Sprintf("Command not found at %d param %d action; ", posParam, posName)
}

func (jsonFile *JSONfiles) createFile(posName, posParam int) string {
	nameOfNewFile, _ := extractNamesFromParamsforActions(jsonFile.JSONfiles[posName].Params[posParam])

	_, err := os.Create(nameOfNewFile)
	if err != nil {
		return fmt.Sprintf("Can't create file %s by %d param %d action + err: %e; ", nameOfNewFile, posParam, posName, err)
	}
	return fmt.Sprintf("File named %s successfully created by %d param %d action;", nameOfNewFile, posParam, posName)
}

func (jsonFile *JSONfiles) renameFile(posName, posParam int) string {
	nameOfOldFile, nameOfNewFile := extractNamesFromParamsforActions(jsonFile.JSONfiles[posName].Params[posParam])

	err := os.Rename(nameOfOldFile, nameOfNewFile)
	if err != nil {
		return fmt.Sprintf("Can't rename file %s to file %s by %d param %d action + err: %e; ", nameOfOldFile, nameOfNewFile, posParam, posName, err)
	}
	return fmt.Sprintf("File renamed from %s to %s successfully by %d param %d action; ", nameOfOldFile, nameOfNewFile, posParam, posName)
}

func (jsonFile *JSONfiles) deleteFile(posName, posParam int) string {
	nameOfDeletedFile, _ := extractNamesFromParamsforActions(jsonFile.JSONfiles[posName].Params[posParam])

	err := os.Remove(nameOfDeletedFile)
	if err != nil {
		return fmt.Sprintf("Can't delete file named %s by %d param %d action + err: %e; ", nameOfDeletedFile, posParam, posName, err)
	}
	return fmt.Sprintf("File named %s successfully deleted by %d param %d action; ", nameOfDeletedFile, posParam, posName)
}

func (jsonFile *JSONfiles) getFileCreationTime(posName, posParam int) string {
	nameOfFile, _ := extractNamesFromParamsforActions(jsonFile.JSONfiles[posName].Params[posParam])

	message, _ := TimeCreationOfFile(nameOfFile, posName, posParam)
	return message
}

func (jsonFile *JSONfiles) writeTextInFile(posName, posParam int) string {
	nameOfFile, textForFile := extractNamesFromParamsforActions(jsonFile.JSONfiles[posName].Params[posParam])

	f, err := os.OpenFile(nameOfFile, os.O_RDWR|os.O_APPEND, os.ModePerm)
	if err != nil {
		return fmt.Sprintf("Unable to open file %s while writing text %s by %d param %d action + err: %e; ", nameOfFile, textForFile, posParam, posName, err)
	}
	defer f.Close()
	_, err = f.WriteString(textForFile)
	if err != nil {
		return fmt.Sprintf("Unable to write text %s in file %s by %d param %d action + err: %e; ", textForFile, nameOfFile, posParam, posName, err)
	}
	return fmt.Sprintf("Text %s successfully written in file %s by %d param %d action; ", textForFile, nameOfFile, posParam, posName)
}

// conditions - это метод, имитирующий принцип работы условия
// Условие сравнивает время создания любого файла (ключевое слово - file) или текущего времени (ключевое слово - current time)
// с указанным в условии в формате DateTime   = "2006-01-02 15:04:05"
// сравнение осуществляется через ключевые слова - "more then", "less then", "equals"
func (jsonFile *JSONfiles) conditions(posName, posParam int) (string, string) {
	param := jsonFile.JSONfiles[posName].Params[posParam]
	nameOfFirstAction, nameOfSecondAction := extractNamesFromParamsforActions(param)
	var firstTimeForComparison, secondTimeForComparison time.Time

	var nameOfFile, errorMessage string
	for i := 0; i < len(param); i++ {
		nameOfFile += string(param[i])
		if strings.HasSuffix(nameOfFile, "current time ") {
			firstTimeForComparison = time.Now()
		}
		if strings.HasSuffix(nameOfFile, "file ") {
			nameOfFile = ""
			for i += 1; string(param[i]) != " "; i++ {
				nameOfFile += string(param[i])
			}
			errorMessage, firstTimeForComparison = TimeCreationOfFile(nameOfFile, posName, posParam)
		}
		if strings.HasSuffix(nameOfFile, "then ") || strings.HasSuffix(nameOfFile, "equals ") {
			nameOfFile = ""
			for i += 1; string(param[i]) != "g"; i++ {
				nameOfFile += string(param[i])
			}
			secondTimeForComparison, _ = time.Parse(time.DateTime, nameOfFile[:len(nameOfFile)-1])
		}

	}
	switch {
	case strings.Contains(param, "more"):
		if firstTimeForComparison.After(secondTimeForComparison) {
			return nameOfFirstAction, fmt.Sprintf("Time %s is More then %s, go to %s by %d param %d condition; ", firstTimeForComparison.Format(time.DateTime), secondTimeForComparison.Format(time.DateTime), nameOfFirstAction, posParam, posName)
		} else {
			return nameOfSecondAction, fmt.Sprintf("Time %s is not More then %s, go to %s by %d param %d condition; ", firstTimeForComparison.Format(time.DateTime), secondTimeForComparison.Format(time.DateTime), nameOfSecondAction, posParam, posName)
		}
	case strings.Contains(param, "less"):
		if firstTimeForComparison.Before(secondTimeForComparison) {
			return nameOfFirstAction, fmt.Sprintf("Time %s is Less then %s, go to %s by %d param %d condition; ", firstTimeForComparison.Format(time.DateTime), secondTimeForComparison.Format(time.DateTime), nameOfFirstAction, posParam, posName)
		} else {
			return nameOfSecondAction, fmt.Sprintf("Time %s is not Less then %s, go to %s by %d param %d condition; ", firstTimeForComparison.Format(time.DateTime), secondTimeForComparison.Format(time.DateTime), nameOfSecondAction, posParam, posName)
		}
	case strings.Contains(param, "equals"):
		if firstTimeForComparison.Equal(secondTimeForComparison) {
			return nameOfFirstAction, fmt.Sprintf("Time %s is Equal then %s, go to %s by %d param %d condition; ", firstTimeForComparison.Format(time.DateTime), secondTimeForComparison.Format(time.DateTime), nameOfFirstAction, posParam, posName)
		} else {
			return nameOfSecondAction, fmt.Sprintf("Time %s is not Equal then %s, go to %s by %d param %d condition; ", firstTimeForComparison.Format(time.DateTime), secondTimeForComparison.Format(time.DateTime), nameOfSecondAction, posParam, posName)
		}
	}
	return fmt.Sprintf("Some erros happend while working with conditions by %d param %d condition err:%s; ", posParam, posName, errorMessage), ""

}

// метод goToNextActionOrCondition - выполняет переход между нужными Действиями/условиями
func (jsonFile *JSONfiles) goToNextActionOrCondition(posName, posParam int, nextAction *string) string {
	_, nameOfFile := extractNamesFromParamsforActions(jsonFile.JSONfiles[posName].Params[posParam])

	*nextAction = nameOfFile

	return fmt.Sprintf("Algorithm successfully switched to %s by %d param %d action; ", nameOfFile, posParam, posName)
}

func main() {
	file, err := os.OpenFile(initFileName, os.O_RDWR, os.ModePerm)
	if err != nil {
		panic(fmt.Errorf("Unable to open file, error: %e", err))
	}

	defer file.Close()

	var jsonFile JSONfiles
	err = jsonFile.readFile(file)
	if err != nil {
		panic(fmt.Errorf("Unable to read from file, error: %e", err))
	}

	var nextAction = "Action1"
	for posName := 0; posName < len(jsonFile.JSONfiles); posName++ {
		resultOfExectuionLog := ""
		if nextAction == jsonFile.JSONfiles[posName].Name {
			for posParam := 0; posParam < len(jsonFile.JSONfiles[posName].Params); posParam++ {
				command, interimLog := jsonFile.extractCommandFromParams(posName, posParam)
				switch command {
				case "Create":
					interimLog = jsonFile.createFile(posName, posParam)
				case "Rename":
					interimLog = jsonFile.renameFile(posName, posParam)
				case "Delete":
					interimLog = jsonFile.deleteFile(posName, posParam)
				case "Get":
					interimLog = jsonFile.getFileCreationTime(posName, posParam)
				case "Write":
					interimLog = jsonFile.writeTextInFile(posName, posParam)
				case "If":
					nextAction, interimLog = jsonFile.conditions(posName, posParam)
				case "Go":
					interimLog = jsonFile.goToNextActionOrCondition(posName, posParam, &nextAction)
				}
				resultOfExectuionLog += interimLog
			}
		}
		err = jsonFile.rewriteFile(posName, resultOfExectuionLog)
		if err != nil {
			panic(fmt.Errorf("Unable to write result of exectuion from file, error: %e", err))
		}
	}

	fmt.Println("Done")
}

func extractNamesFromParamsforActions(commandString string) (nameOfFirstFile, nameOfSecondFileOrText string) {
	var nameOfFile string
	var counter int
	for i := 0; i < len(commandString); i++ {
		nameOfFile += string(commandString[i])
		if (strings.HasSuffix(nameOfFile, "file ") && !strings.HasPrefix(commandString, "If")) || (strings.HasSuffix(nameOfFile, "to ") && strings.HasPrefix(commandString, "If")) {
			nameOfFile = ""

			for i += 1; i < len(commandString) && !strings.HasSuffix(nameOfFile, " "); i++ {
				nameOfFile += string(commandString[i])
			}
			counter++
			switch counter {
			case 1:
				nameOfFirstFile = nameOfFile
				if strings.HasSuffix(nameOfFile, " ") {
					nameOfFirstFile = nameOfFile[:len(nameOfFile)-1]
					i--
				}
			case 2:
				nameOfSecondFileOrText = nameOfFile
			}
		}
		if strings.HasSuffix(nameOfFile, "text ") || strings.HasSuffix(nameOfFile, "to ") {
			nameOfSecondFileOrText = commandString[i+1:]
		}
	}
	return nameOfFirstFile, nameOfSecondFileOrText
}

func TimeCreationOfFile(nameOfFile string, posName, posParam int) (string, time.Time) {
	stat, err := os.Stat(nameOfFile)
	if err != nil {
		return fmt.Sprintf("Can't extract time of file named %s by %d param %d action + err: %e; ", nameOfFile, posParam, posName, err), time.Time{}
	}
	os := runtime.GOOS
	switch os {
	/*case "windows":
	OSstat := stat.Sys().(*syscall.Win32FileAttributeData)
	ctime = time.Unix(0, OSstat.CreationTime.Nanoseconds())
	return fmt.Sprintf("Time creation of file %s is: %s at OS %s by %d param %d action; ", nameOfFile, ctime, os, posParam, posName), ctime
	*/
	// В случае если ОС Windows следует убрать комментарии у case Windows и добавить комментарии для UNIX-подобных систем
	case "darwin":
	case "linux":
		OSstat := stat.Sys().(*syscall.Stat_t)
		ctime := time.Unix(OSstat.Ctim.Sec, 0)
		return fmt.Sprintf("Time creation of file %s is: %s at OS %s by %d param %d action; ", nameOfFile, ctime, os, posParam, posName), ctime
	}
	return fmt.Sprintf("Unable to find file creation time at OS %s of file named %s by %d param %d action; ", os, nameOfFile, posParam, posName), time.Time{}
	//Программа работает только с Windows,Linux и Mac,но проверку на создание времени файла можно добавить и для других ОС
	//в частности идеи для проверки времени подсмотрены тут - https://github.com/djherbis/times
}
