package main

import (
	"testing"
)

func TestRGB(t *testing.T) {
	// test Parse()
	parseTestRGB := &RGB{}
	type parseTest struct {
		s        string
		expected RGB
	}
	// Parse tests that should pass
	for _, test := range []parseTest{
		{"#123", RGB{[3]uint8{0x11, 0x22, 0x33}}},
		{"#Abc", RGB{[3]uint8{0xaa, 0xbb, 0xcc}}},
		{"#123456", RGB{[3]uint8{0x12, 0x34, 0x56}}},
		{"#abcDEF", RGB{[3]uint8{0xab, 0xcd, 0xef}}},
		{"1e3", RGB{[3]uint8{0x11, 0xee, 0x33}}},
		{"842AB0", RGB{[3]uint8{0x84, 0x2a, 0xb0}}},
	} {
		if err := parseTestRGB.Parse(test.s); err != nil {
			t.Errorf("Parse(%s) on %v returned an error instead of a valid result: %v", test.s, parseTestRGB, err)
			continue
		}
		if *parseTestRGB != test.expected {
			t.Errorf("Parse(%s) result did not match %v. Got %v", test.s, test.expected, parseTestRGB)
		}
	}
	// Parse tests that should fail
	for _, test := range []parseTest{
		{"", RGB{}},
		{"#", RGB{}},
		{"#1", RGB{}},
		{"#12", RGB{}},
		{"#1234", RGB{}},
		{"#12345", RGB{}},
		{"#1234567", RGB{}},
		{"#abcDEX", RGB{}},
		{"!1e3", RGB{}},
		{"842AB0?", RGB{}},
	} {
		if err := parseTestRGB.Parse(test.s); err == nil || err.Error() != "Invalid RGB string" {
			t.Errorf("Parse(%s) on %v did not return the expected error: %v", test.s, parseTestRGB, err)
			continue
		}
	}

	// test IsNearDuplicate()
	rgb1 := &RGB{[3]uint8{50, 100, 150}} // diff = 5 + 8 + 9 = 22
	rgb2 := &RGB{[3]uint8{55, 92, 159}}
	for _, maxDiff := range []int{1, 8, 21} {
		if rgb1.IsNearDuplicate(rgb2, maxDiff) == true {
			t.Errorf("IsNearDuplicate with tolerance=%d returned true (expected false) for %v and %v", maxDiff, rgb1, rgb2)
		}
		if rgb2.IsNearDuplicate(rgb1, maxDiff) == true {
			t.Errorf("IsNearDuplicate with tolerance=%d returned true (expected false) for %v and %v", maxDiff, rgb2, rgb1)
		}
	}
	for _, maxDiff := range []int{22, 23, 255} {
		if rgb1.IsNearDuplicate(rgb2, maxDiff) == false {
			t.Errorf("IsNearDuplicate with tolerance=%d returned false (expected true) for %v and %v", maxDiff, rgb1, rgb2)
		}
		if rgb2.IsNearDuplicate(rgb1, maxDiff) == false {
			t.Errorf("IsNearDuplicate with tolerance=%d returned false (expected true) for %v and %v", maxDiff, rgb2, rgb1)
		}
	}
	if rgb1.IsNearDuplicate(rgb1, 0) != true {
		t.Errorf("IsNearDuplicate for equal items did not return true for %v", rgb1)
	}

	// test RandomizeAvoidingDuplicates()
	toRandomize := &RGB{}
	onlyRedFreeDelta126 := []*RGB{ // only 0x80 free with tolerance 126
		{[3]uint8{0x01, 0x22, 0x33}},
		{[3]uint8{0xff, 0xaa, 0xbb}},
	}
	onlyGreenFreeDelta100 := []*RGB{ // only 154 free with tolerance 100
		{[3]uint8{50, 25, 51}},
		{[3]uint8{125, 53, 126}},
		{[3]uint8{200, 255, 201}},
	}
	onlyBlueFreeDelta50 := []*RGB{ // only 100 free with tolerance 50
		{[3]uint8{25, 25, 25}},
		{[3]uint8{75, 75, 49}},
		{[3]uint8{125, 125, 151}},
		{[3]uint8{175, 175, 175}},
		{[3]uint8{225, 225, 225}},
		{[3]uint8{250, 250, 250}},
	}
	var err error

	err = toRandomize.RandomizeAvoidingDuplicates(126, onlyRedFreeDelta126)
	if err != nil {
		t.Errorf("RandomizeAvoidingDuplicates returned unexpected error: %v", err)
	}
	if toRandomize.rgb[0] != 0x80 {
		t.Errorf("RandomizeAvoidingDuplicates expected red=%x. Got %v", 0x80, toRandomize)
	}
	t.Log("onlyRedFree result:", toRandomize)

	err = toRandomize.RandomizeAvoidingDuplicates(100, onlyGreenFreeDelta100)
	if err != nil {
		t.Errorf("RandomizeAvoidingDuplicates returned unexpected error: %v", err)
	}
	if toRandomize.rgb[1] != 154 {
		t.Errorf("RandomizeAvoidingDuplicates expected green=%x. got %v", 154, toRandomize)
	}
	t.Log("onlyGreenFree result:", toRandomize)

	err = toRandomize.RandomizeAvoidingDuplicates(50, onlyBlueFreeDelta50)
	if err != nil {
		t.Errorf("RandomizeAvoidingDuplicates returned unexpected error: %v", err)
	}
	if toRandomize.rgb[2] != 100 {
		t.Errorf("RandomizeAvoidingDuplicates expected blue=%x. got %v", 100, toRandomize)
	}
	t.Log("onlyBlueFree result:", toRandomize)

	avoid1 := []*RGB{
		{[3]uint8{0x00, 0x00, 0x00}},
		{[3]uint8{0x10, 0x10, 0x10}},
		{[3]uint8{0x20, 0x20, 0x20}},
		{[3]uint8{0x30, 0x30, 0x30}},
		{[3]uint8{0x40, 0x40, 0x40}},
		{[3]uint8{0x50, 0x50, 0x50}},
		{[3]uint8{0x60, 0x60, 0x60}},
		{[3]uint8{0x70, 0x70, 0x70}},
		{[3]uint8{0x80, 0x80, 0x80}},
		{[3]uint8{0x90, 0x90, 0x90}},
	}
	avoid2 := []*RGB{
		{[3]uint8{0xa0, 0xa0, 0xa0}},
		{[3]uint8{0xb0, 0xb0, 0xb0}},
		{[3]uint8{0xc0, 0xc0, 0xc0}},
		{[3]uint8{0xd0, 0xd0, 0xd0}},
		{[3]uint8{0xe0, 0xe0, 0xe0}},
		{[3]uint8{0xf0, 0xf0, 0xf0}},
		{[3]uint8{0xff, 0xff, 0xff}},
	}
	for tolerance := 0; tolerance < 8; tolerance++ {
		err = toRandomize.RandomizeAvoidingDuplicates(tolerance, avoid1, avoid2)
		if err != nil {
			t.Errorf("RandomizeAvoidingDuplicates returned unexpected error for tolerance=%d: %v", tolerance, err)
		}
	}
	for tolerance := 8; tolerance < 16; tolerance++ {
		err = toRandomize.RandomizeAvoidingDuplicates(tolerance, avoid1, avoid2)
		if err == nil || err.Error() != "No available colors match the tolerance constraint" {
			t.Errorf("RandomizeAvoidingDuplicates should have return a 'No available colors' error but didn't. err=%v", err)
		}
	}
}

func BenchmarkRandomizeAvoidingDuplicates(b *testing.B) {
	toRandomize := &RGB{}
	avoid1 := []*RGB{
		{[3]uint8{0xe1, 0x00, 0x41}},
		{[3]uint8{0x50, 0xf2, 0xec}},
		{[3]uint8{0xdf, 0x7e, 0xd3}},
		{[3]uint8{0xae, 0x43, 0x2c}},
		{[3]uint8{0x11, 0x11, 0x11}},
		{[3]uint8{0x88, 0x88, 0x88}},
	}
	avoid2 := []*RGB{
		{[3]uint8{0x21, 0x36, 0x6e}},
		{[3]uint8{0x34, 0x92, 0x9f}},
		{[3]uint8{0x69, 0xcc, 0xcd}},
	}
	var err error

	for b.Loop() {
		err = toRandomize.RandomizeAvoidingDuplicates(16, avoid1, avoid2)
		if err != nil {
			b.Fatal("An error was returned from RandomizeAvoidingDuplicates:", err)
		}

	}
}
