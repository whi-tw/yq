package yqlib

import (
	"testing"
)

var dateTimeOperatorScenarios = []expressionScenario{
	{
		description:    "Format: from ISO standard date time",
		subdescription: "Providing a single parameter assumes a standard ISO datetime format. If the target format is not a valid yaml datetime format, the result will be a string tagged node.",
		document:       `a: 2001-12-15T02:59:43.1Z`,
		expression:     `.a |= format_datetime("Monday, 02-Jan-06 at 3:04PM")`,
		expected: []string{
			"D0, P[], (doc)::a: Saturday, 15-Dec-01 at 2:59AM\n",
		},
	},
	{
		description:    "Format: from custom date time",
		subdescription: "if the target format is a valid yaml datetime format, then it will automatically get the !!timestamp tag.",
		document:       `a: Saturday, 15-Dec-01 at 2:59AM`,
		expression:     `.a |= format_datetime("Monday, 02-Jan-06 at 3:04PM"; "2006-01-02")`,
		expected: []string{
			"D0, P[], (doc)::a: 2001-12-15\n",
		},
	},
	{
		description: "Now",
		document:    "a: cool",
		expression:  `.updated = now`,
		expected: []string{
			"D0, P[], (doc)::a: cool\nupdated: 2021-05-19T01:02:03Z\n",
		},
	},
}

func TestDatetimeOperatorScenarios(t *testing.T) {
	for _, tt := range dateTimeOperatorScenarios {
		testScenario(t, &tt)
	}
	documentOperatorScenarios(t, "datetime", dateTimeOperatorScenarios)
}
