package main

import (
	"testing"
)

func TestRemoveBadWords(t *testing.T) {
	message := "This has kerfuffle in it"
	result := replaceBadWord(message)
	expected := "This has **** in it"
	if result != expected {
		t.Errorf("Result: %v; Expected: %v", result, expected)
	}
}

func TestRemoveBadWords2(t *testing.T) {
	message := "This has sharbert in it"
	result := replaceBadWord(message)
	expected := "This has **** in it"
	if result != expected {
		t.Errorf("Result: %v; Expected: %v", result, expected)
	}
}

func TestRemoveBadWords3(t *testing.T) {
	message := "This has fornax in it"
	result := replaceBadWord(message)
	expected := "This has **** in it"
	if result != expected {
		t.Errorf("Result: %v; Expected: %v", result, expected)
	}
}

func TestRemoveBadWords4(t *testing.T) {
	message := "This has Fornax in it"
	result := replaceBadWord(message)
	expected := "This has **** in it"
	if result != expected {
		t.Errorf("Result: %v; Expected: %v", result, expected)
	}
}

func TestRemoveBadWords5(t *testing.T) {
	message := "This has SHARBERT in it"
	result := replaceBadWord(message)
	expected := "This has **** in it"
	if result != expected {
		t.Errorf("Result: %v; Expected: %v", result, expected)
	}
}
