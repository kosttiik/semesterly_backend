// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/get-data": {
            "get": {
                "description": "Возвращает данные расписания из базы данных в формате JSON",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "GetData"
                ],
                "summary": "Получение расписания",
                "responses": {
                    "200": {
                        "description": "Список элементов расписания",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/models.ScheduleItem"
                            }
                        }
                    },
                    "500": {
                        "description": "error: Failed to fetch schedule items",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/get-group-schedule/{uuid}": {
            "get": {
                "description": "Возвращает данные расписания конкретной группы из базы данных в формате JSON",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "GetData"
                ],
                "summary": "Получение расписания группы",
                "parameters": [
                    {
                        "type": "string",
                        "description": "UUID группы",
                        "name": "uuid",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "Список элементов расписания",
                        "schema": {
                            "type": "array",
                            "items": {
                                "$ref": "#/definitions/models.ScheduleItem"
                            }
                        }
                    },
                    "500": {
                        "description": "error: Failed to fetch schedule items",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        },
        "/hello": {
            "get": {
                "description": "Проверяет, работает ли сервер и есть ли подключение к базе данных",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "Hello"
                ],
                "summary": "Проверка подключения",
                "responses": {
                    "200": {
                        "description": "Hello, World!",
                        "schema": {
                            "type": "string"
                        }
                    }
                }
            }
        },
        "/insert-data": {
            "post": {
                "description": "Вставляет данные расписания и экзаменов в базу данных",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "InsertData"
                ],
                "summary": "Вставка данных",
                "responses": {
                    "200": {
                        "description": "message: Data inserted successfully",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "errors: [error messages]",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/insert-group-schedule/{uuid}": {
            "post": {
                "description": "Вставляет данные расписания и экзаменов для конкретной группы в базу данных",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "InsertGroupSchedule"
                ],
                "summary": "Вставка расписания группы",
                "parameters": [
                    {
                        "type": "string",
                        "description": "UUID группы",
                        "name": "uuid",
                        "in": "path",
                        "required": true
                    }
                ],
                "responses": {
                    "200": {
                        "description": "message: Group schedule inserted successfully",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "errors: [error messages]",
                        "schema": {
                            "type": "object",
                            "additionalProperties": true
                        }
                    }
                }
            }
        },
        "/write-schedule": {
            "post": {
                "description": "Сохраняет данные расписания в CSV файл",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "tags": [
                    "WriteSchedule"
                ],
                "summary": "Сохранение расписания",
                "responses": {
                    "200": {
                        "description": "message: Schedule written to file successfully",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    },
                    "500": {
                        "description": "error: Failed to fetch schedule items\" \"error: Failed to write schedule to file",
                        "schema": {
                            "type": "object",
                            "additionalProperties": {
                                "type": "string"
                            }
                        }
                    }
                }
            }
        }
    },
    "definitions": {
        "models.Audience": {
            "type": "object",
            "properties": {
                "building": {
                    "type": "string"
                },
                "created_at": {
                    "type": "string"
                },
                "deleted_at": {
                    "type": "string"
                },
                "department_uid": {
                    "description": "Может быть null",
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "name": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                },
                "uuid": {
                    "type": "string"
                }
            }
        },
        "models.Discipline": {
            "type": "object",
            "properties": {
                "abbr": {
                    "type": "string"
                },
                "actType": {
                    "type": "string"
                },
                "fullName": {
                    "type": "string"
                },
                "shortName": {
                    "type": "string"
                }
            }
        },
        "models.Group": {
            "type": "object",
            "properties": {
                "created_at": {
                    "type": "string"
                },
                "deleted_at": {
                    "type": "string"
                },
                "department_uid": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "name": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                },
                "uuid": {
                    "type": "string"
                }
            }
        },
        "models.ScheduleItem": {
            "type": "object",
            "properties": {
                "audiences": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/models.Audience"
                    }
                },
                "created_at": {
                    "type": "string"
                },
                "day": {
                    "type": "integer"
                },
                "deleted_at": {
                    "type": "string"
                },
                "discipline": {
                    "$ref": "#/definitions/models.Discipline"
                },
                "endTime": {
                    "type": "string"
                },
                "groups": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/models.Group"
                    }
                },
                "id": {
                    "type": "integer"
                },
                "permission": {
                    "type": "string"
                },
                "startTime": {
                    "type": "string"
                },
                "stream": {
                    "type": "string"
                },
                "teachers": {
                    "type": "array",
                    "items": {
                        "$ref": "#/definitions/models.Teacher"
                    }
                },
                "time": {
                    "type": "integer"
                },
                "updated_at": {
                    "type": "string"
                },
                "week": {
                    "type": "string"
                }
            }
        },
        "models.Teacher": {
            "type": "object",
            "properties": {
                "created_at": {
                    "type": "string"
                },
                "deleted_at": {
                    "type": "string"
                },
                "firstName": {
                    "type": "string"
                },
                "id": {
                    "type": "integer"
                },
                "lastName": {
                    "type": "string"
                },
                "middleName": {
                    "type": "string"
                },
                "updated_at": {
                    "type": "string"
                },
                "uuid": {
                    "type": "string"
                }
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "1.0",
	Host:             "localhost:8080",
	BasePath:         "/api/v1",
	Schemes:          []string{},
	Title:            "Автоматизированная система по ведению расписания учебных занятий",
	Description:      "API для управления расписанием учебных занятий",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
