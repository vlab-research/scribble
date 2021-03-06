package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	surveySql = `drop table if exists surveys;
                 create table if not exists surveys(
                   userid VARCHAR NOT NULL,
                   id UUID NOT NULL UNIQUE,
                   form_json JSON,
                   created TIMESTAMPTZ NOT NULL,
                   translation_conf JSON
                 );`

	insertSurveySql = `INSERT INTO surveys(userid, created, id, form_json, translation_conf) VALUES ('owner', NOW(), $1, $2, $3);`

	formA = `{"fields": [
          {"title": "What is your gender? ",
           "ref": "eng_foo",
           "properties": {
              "choices": [{"label": "Male"},
                          {"label": "Female"},
                          {"label": "Other"}]},
           "type": "multiple_choice"},
          {"title": "Which state do you currently live in?\n- A. foo 91  bar\n- B. Jharkhand\n- C. Odisha\n- D. Uttar Pradesh",
           "ref": "eng_bar",
           "properties": {"choices": [{"label": "A"},
                                      {"label": "B"},
                                      {"label": "C"},
                                      {"label": "D"}]},
           "type": "multiple_choice"},
           {"title": "How old are you?",
           "ref": "eng_baz",
           "properties": {},
           "type": "number"}]}`

	formB = `{"title": "mytitle", "fields": [
          {"id": "vjl6LihKMtcX",
          "title": "आपका लिंग क्या है? ",
          "ref": "foo",
          "properties": {"choices": [{"label": "पुरुष"},
                                    {"label": "महिला"},
                                    {"label": "अन्य"}]},
          "type": "multiple_choice"},
          {"id": "mdUpJMSY8Lct",
           "title": "वर्तमान में आप किस राज्य में रहते हैं?\n- A. छत्तीसगढ़\n- B. झारखंड\n- C. ओडिशा\n- D. उत्तर प्रदेश",
           "ref": "bar",
           "properties": {"choices": [{"label": "A"},
                                      {"label": "B"},
                                      {"label": "C"},
                                      {"label": "D"}]},
           "type": "multiple_choice"},
          {"id": "mdUpJMSY8Lct",
           "title": "वर्तमान में आप किस राज्य में रहते हैं?",
           "ref": "baz",
           "properties": {},
           "type": "number"}]}`

	responseSql = `drop table if exists surveys;
                   drop table if exists responses;
            create table if not exists responses(
			  parent_shortcode VARCHAR NOT NULL,
			  surveyid UUID NOT NULL,
			  shortcode VARCHAR NOT NULL,
			  flowid INT NOT NULL,
			  userid VARCHAR NOT NULL,
			  pageid VARCHAR,
			  question_ref VARCHAR NOT NULL,
			  question_idx INT NOT NULL,
			  question_text VARCHAR NOT NULL,
			  response VARCHAR NOT NULL,
			  translated_response VARCHAR,
			  seed INT NOT NULL,
			  timestamp TIMESTAMPTZ NOT NULL,
              metadata JSONB,
              CONSTRAINT "valid_metadata" CHECK (json_typeof(metadata) = 'object'),
			  PRIMARY KEY (userid, timestamp, question_ref)
           );`
)

func TestResponseWriterWritesGoodData(t *testing.T) {
	pool := testPool()
	defer pool.Close()

	mustExec(t, pool, responseSql)

	msgs := makeMessages([]string{
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"foo",
          "pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":"LOL",
          "seed":858044518,
          "metadata": {"foo":"bar","seed": 8978437},
          "timestamp":1599039840517}`,
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"bar",
          "pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":"LOL",
          "seed":858044518,
          "metadata": {"foo":"bar","seed": 8978437},
          "timestamp":1599039840517}`,
	})

	writer := GetWriter(NewResponseScribbler(pool))
	err := writer.Write(msgs)
	assert.Nil(t, err)

	res := getCol(pool, "responses", "userid")
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "bar", *res[0])
	assert.Equal(t, "foo", *res[1])

	res = getCol(pool, "responses", "metadata->>'foo'")
	assert.Equal(t, 2, len(res))
	assert.Equal(t, "bar", *res[0])
	assert.Equal(t, "bar", *res[1])

	mustExec(t, pool, "drop table responses")
}

func TestResponseWriterWritesNullPageIdIfNone(t *testing.T) {
	pool := testPool()
	defer pool.Close()

	mustExec(t, pool, responseSql)

	msgs := makeMessages([]string{
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"foo",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":"LOL",
          "seed":858044518,
          "metadata": {"foo":"bar","seed": 8978437},
          "timestamp":1599039840517}`,
	})

	writer := GetWriter(NewResponseScribbler(pool))
	err := writer.Write(msgs)
	assert.Nil(t, err)

	rows, err := pool.Query(context.Background(), "select pageid from responses where pageid is null")

	res := rowStrings(rows)
	assert.Equal(t, 1, len(res))
	mustExec(t, pool, "drop table responses")
}

func TestResponseWriterWritesPageIdIfExists(t *testing.T) {
	pool := testPool()
	defer pool.Close()

	mustExec(t, pool, responseSql)

	msgs := makeMessages([]string{
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"foo",
          "pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":"LOL",
          "seed":858044518,
          "metadata": {"foo":"bar","seed": 8978437},
          "timestamp":1599039840517}`,
	})

	writer := GetWriter(NewResponseScribbler(pool))
	err := writer.Write(msgs)
	assert.Nil(t, err)

	res := getCol(pool, "responses", "pageid")
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "baz", *res[0])

	mustExec(t, pool, "drop table responses")
}

func TestResponseWriterHandlesMixedResponseAndShortCodeTypes(t *testing.T) {
	pool := testPool()
	defer pool.Close()

	mustExec(t, pool, responseSql)

	msgs := makeMessages([]string{
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"foo",
          "pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":true,
          "seed":858044518,
          "metadata": {"foo":"bar","seed": 8978437},
          "timestamp":1599039840517}`,
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"foo",
          "pageid": "baz",
          "question_ref":"baz",
          "question_idx":1,
          "question_text":"foobar",
          "response":"yes",
          "seed":858044518,
          "metadata": {"foo":"bar","seed": 8978437},
          "timestamp":1599039840517}`,
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":123,
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":123,
          "flowid":1,
          "userid":"foo",
          "pageid": "baz",
          "question_ref":"qux",
          "question_idx":1,
          "question_text":"foobar",
          "response":25,
          "seed":858044518,
          "metadata": {"foo":"bar","seed": 8978437},
          "timestamp":1599039840517}`,
	})

	writer := GetWriter(NewResponseScribbler(pool))
	err := writer.Write(msgs)
	assert.Nil(t, err)

	res := getCol(pool, "responses", "response")
	assert.Equal(t, 3, len(res))
	assert.Equal(t, "true", *res[0])
	assert.Equal(t, "yes", *res[1])
	assert.Equal(t, "25", *res[2])

	res = getCol(pool, "responses", "shortcode")
	assert.Equal(t, 3, len(res))
	assert.Equal(t, "baz", *res[0])
	assert.Equal(t, "123", *res[2])

	mustExec(t, pool, "drop table responses")
}

func TestResponseWriterFailsOnMissingData(t *testing.T) {
	pool := testPool()
	defer pool.Close()

	mustExec(t, pool, responseSql)

	msgs := makeMessages([]string{
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"foo",
"pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "seed":858044518,
          "metadata": {"foo":"bar","seed": 8978437},
          "timestamp":1599039840517}`,
	})

	writer := GetWriter(NewResponseScribbler(pool))
	err := writer.Write(msgs)
	assert.NotNil(t, err)

	res := getCol(pool, "responses", "userid")
	assert.Equal(t, 0, len(res))
	mustExec(t, pool, "drop table responses")
}

func TestResponseWriterFailsOnMissingMetadata(t *testing.T) {
	pool := testPool()
	defer pool.Close()

	mustExec(t, pool, responseSql)

	msgs := makeMessages([]string{
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"foo",
          "pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":"LOL",
          "seed":858044518,
          "timestamp":1599039840517}`,
	})

	writer := GetWriter(NewResponseScribbler(pool))
	err := writer.Write(msgs)
	assert.NotNil(t, err)

	res := getCol(pool, "responses", "userid")
	assert.Equal(t, 0, len(res))
	mustExec(t, pool, "drop table responses")
}

func TestResponseWriterFailsIfMetadataFormatedPoorly(t *testing.T) {
	pool := testPool()
	defer pool.Close()

	mustExec(t, pool, responseSql)

	msgs := makeMessages([]string{
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"foo",
          "pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":"LOL",
          "seed":858044518,
          "metadata":"{\"foo\":\"bar\",\"startTime\":1599039840517}",
          "timestamp":1599039840517}`,
	})

	writer := GetWriter(NewResponseScribbler(pool))
	err := writer.Write(msgs)
	assert.NotNil(t, err)

	res := getCol(pool, "responses", "metadata->>'foo'")
	assert.Equal(t, 0, len(res))

	mustExec(t, pool, "drop table responses")
}

func TestResponseWriterSucceedsIfMetadataEmpty(t *testing.T) {
	pool := testPool()
	defer pool.Close()

	mustExec(t, pool, responseSql)

	msgs := makeMessages([]string{
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"foo",
          "pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":"LOL",
          "seed":858044518,
          "metadata":{},
          "timestamp":1599039840517}`,
	})

	writer := GetWriter(NewResponseScribbler(pool))
	err := writer.Write(msgs)
	assert.Nil(t, err)

	res := getCol(pool, "responses", "metadata->>'foo'")
	assert.Equal(t, 1, len(res))

	mustExec(t, pool, "drop table responses")
}

func TestResponseWriterIgnoresRepeatMessages(t *testing.T) {
	pool := testPool()
	defer pool.Close()

	mustExec(t, pool, responseSql)

	msgs := makeMessages([]string{
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"foo",
          "pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":"LOL",
          "seed":858044518,
          "metadata": {"foo":"bar","seed": 8978437},
          "timestamp":1599039840517}`,
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"foo",
          "pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":"LOL",
          "seed":858044518,
          "metadata": {"foo":"bar","seed": 8978437},
          "timestamp":1599039840517}`,
	})

	writer := GetWriter(NewResponseScribbler(pool))
	err := writer.Write(msgs)
	assert.Nil(t, err)

	res := getCol(pool, "responses", "userid")
	assert.Equal(t, 1, len(res))
	assert.Equal(t, "foo", *res[0])

	mustExec(t, pool, "drop table responses")
}

func TestResponseWriterTranslatesSuccesfullyToOtherForm(t *testing.T) {
	pool := testPool()
	defer pool.Close()

	mustExec(t, pool, responseSql)
	mustExec(t, pool, surveySql)
	mustExec(t, pool, insertSurveySql, "25d88630-8b7b-4f2b-8630-4e5f9085e888", formA, `{}`)
	mustExec(t, pool, insertSurveySql, "d6c21c81-fcd0-4aa4-8975-8584d8bdb820", formB, `{"destination": "25d88630-8b7b-4f2b-8630-4e5f9085e888"}`)

	msgs := makeMessages([]string{
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"bar",
          "pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":"LOL",
          "seed":858044518,
          "metadata":{},
          "timestamp":1599039840517}`,
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"bar",
          "pageid": "baz",
          "question_ref":"bar",
          "question_idx":1,
          "question_text":"foobar",
          "response":"A",
          "seed":858044518,
          "metadata":{},
          "timestamp":1999099840999}`,
		`{"parent_surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "parent_shortcode":"baz",
          "surveyid":"d6c21c81-fcd0-4aa4-8975-8584d8bdb820",
          "shortcode":"baz",
          "flowid":1,
          "userid":"bar",
          "pageid": "baz",
          "question_ref":"foo",
          "question_idx":1,
          "question_text":"foobar",
          "response":"अन्य",
          "seed":858044518,
          "metadata":{},
          "timestamp":1999099840999}`,
	})

	writer := GetWriter(NewResponseScribbler(pool))
	err := writer.Write(msgs[:1])
	assert.Nil(t, err)

	err = writer.Write(msgs[1:])
	assert.Nil(t, err)

	res := getCol(pool, "responses", "response")
	assert.Equal(t, 3, len(res))
	assert.Equal(t, "LOL", *res[0])
	assert.Equal(t, "A", *res[1])
	assert.Equal(t, "अन्य", *res[2])

	res = getCol(pool, "responses", "translated_response")
	assert.Equal(t, 3, len(res))

	assert.Nil(t, res[0])
	assert.Equal(t, "foo 91  bar", *res[1])
	assert.Equal(t, "Other", *res[2])

	mustExec(t, pool, "drop table responses")
}
