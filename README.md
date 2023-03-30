# TestProjectWithJSON
From conditions to action to conditions

Код выполняет определённые команды:
     
        a. Создать файл
        b. Изменить название файла
        c. Удалить файл 
        d. Получить время создания файла
        e. Запись произвольной строки в файл
        f. Условие – дата и время больше <значение> 

В файл Actions.json - записывается JSON - структура, на основе которой происходит выполнение действий/условий, а также куда записывается результат выполнения условий/действий. У каждого условия и действия есть своё название, по которому осуществляется переход от одного действия или условия к другому

В ExampleJSONFile.json можно в удобном формате записывать изначальную структуру, которую можно будет копировать в Actions.json, код работает именно со структурой Actions.json. Также в ExampleJSONFile.json записаны примеры всех команд в разных комбинациях условий и действий, в соответсвии с инструкциями можно прописывать свои услвоия и действия в любом удобной форме соблюдая правила оформления структуры и записи массивов параметров

Структура JSON представляет с собой:

type JSONfiles struct {
	JSONfiles []JSONfile `json:"JSONfiles"`
}
JSONfiles позволяет использовать внутри JSON несколько подобных структур

type JSONfile struct {
	Name              string   `json:"name"` - Название Действия/Условия
	ResultOfExectuion string   `json:"resultOfExectuion"` - Результат выполнения последовательных параметров внутри действия, разделённых - ";"
	Params            []string `json:"params"` - Перечисленные парметры, в которых прописаны команды и другие необходимые атрибуты
}

Вместе со структурой реализован интерфейс:

type interactionsWithJSONFile interface {
	readFile(fileName *os.File, jsonFile *JSONfiles) (JSONfiles, error) - Чтение файла для получения параметров структуры JSON
	rewriteFile(fileName *os.File, pos int, resultText string) error - Перезаписывание поля resultOfExectuion - для описание результата последовательных действий
	extractCommandFromParams(posName, posParam int) (string, string) - Извлечение из параметров ключевого слова команды
	createFile(posName, posParam int) string - Создание файла
	renameFile(posName, posParam int) string - Переименовывание файла
	deleteFile(posName, posParam int) string - Удаление файла
	getFileCreationTime(posName, posParam int) string - получение времени создания файла
	writeTextInFile(posName, posParam int) string - запись произвольного текста в файл
	conditions(posName, posParam int) (string, string) - реализация поведения условия
	goToNextActionOrCondition(posName, posParam int, nextAction *string) string - переход от Условий и Действий к другим Условиям и Действиям
}

В ProjectWithJSON.go находится код, так же там есть комментарии к определённым участкам кода
