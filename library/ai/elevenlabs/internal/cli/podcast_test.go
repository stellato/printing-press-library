package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePodcastScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "episode.md")
	script := `---
title: "The Focus Premium"
episode: 7
language: en
model: eleven_v3
output_format: mp3_44100_192
loudness: -16
cast:
  HOST: Rachel
  GUEST: Antoni
music:
  intro: { prompt: "warm intro", seconds: 12 }
  outro: { prompt: "soft outro", seconds: 10 }
  bed: { prompt: "ambient bed", duck_db: -15 }
---

[intro]

HOST: Welcome back.
GUEST: Glad to be here.

[music: bed]
HOST: Let's start.
[sfx: page turn, 1.5s]
GUEST: Perfect.
[music: stop]

[outro]
`
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	episode, err := parsePodcastScript(path)
	if err != nil {
		t.Fatal(err)
	}
	if episode.Title != "The Focus Premium" {
		t.Fatalf("title = %q", episode.Title)
	}
	if episode.TextChars == 0 {
		t.Fatal("expected text chars")
	}
	if got := len(episode.Segments); got != 6 {
		t.Fatalf("segments = %d", got)
	}
	if episode.Segments[2].BedName != "bed" {
		t.Fatalf("bed segment = %q", episode.Segments[2].BedName)
	}
	if episode.Segments[3].Kind != "sfx" || episode.Segments[3].SFXSeconds != 1.5 {
		t.Fatalf("unexpected sfx segment: %+v", episode.Segments[3])
	}
}

func TestParsePodcastScriptUnknownSpeaker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.md")
	script := `---
cast:
  HOST: Rachel
---
GUEST: Hello.
`
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := parsePodcastScript(path)
	if err == nil || !strings.Contains(err.Error(), "not in cast") {
		t.Fatalf("expected cast error, got %v", err)
	}
}

func TestParseLoudnormMeasurement(t *testing.T) {
	output := `frame=1
{
	"input_i" : "-21.72",
	"input_tp" : "-3.14",
	"input_lra" : "8.20",
	"input_thresh" : "-31.80",
	"output_i" : "-16.01",
	"target_offset" : "0.02"
}`
	m, err := parseLoudnormMeasurement(output)
	if err != nil {
		t.Fatal(err)
	}
	if m.TargetOffset != "0.02" {
		t.Fatalf("offset = %q", m.TargetOffset)
	}
}

func TestBuildPodcastFilterGraphNormalizesAndDucks(t *testing.T) {
	graph := buildPodcastFilterGraph([]podcastMixItem{
		{Kind: "intro", Path: "intro.mp3", DurationSeconds: 3},
		{Kind: "voice", Path: "voice.mp3", BedPath: "bed.mp3", DurationSeconds: 4},
		{Kind: "outro", Path: "outro.mp3", DurationSeconds: 3},
	})
	for _, want := range []string{
		"aresample=44100,aformat=sample_fmts=fltp:channel_layouts=stereo",
		"aloop=loop=-1:size=176400",
		"sidechaincompress=threshold=0.03",
		"acrossfade=d=1.0",
	} {
		if !strings.Contains(graph, want) {
			t.Fatalf("filter graph missing %q:\n%s", want, graph)
		}
	}
}

func TestPodcastMasterVariants(t *testing.T) {
	variants := podcastMasterVariants(podcastMasterOptions{
		Out:        "episode.mp3",
		TargetLUFS: -16,
		TruePeak:   -1,
		Variants:   "apple,spotify",
	})
	if len(variants) != 2 {
		t.Fatalf("variants = %d", len(variants))
	}
	if variants[0].Path != "episode-apple.mp3" || variants[0].TargetLUFS != -16 {
		t.Fatalf("apple variant = %+v", variants[0])
	}
	if variants[1].Path != "episode-spotify.mp3" || variants[1].TargetLUFS != -14 {
		t.Fatalf("spotify variant = %+v", variants[1])
	}
}

func TestLoudnormPass(t *testing.T) {
	if !loudnormPass(loudnormMeasurement{InputI: "-16.1", InputTP: "-1.2"}, -16, -1) {
		t.Fatal("expected loudnorm pass")
	}
	if loudnormPass(loudnormMeasurement{InputI: "-18.5", InputTP: "-0.1"}, -16, -1) {
		t.Fatal("expected loudnorm failure")
	}
}

func TestBuildPodcastSEOAssets(t *testing.T) {
	transcript := "HOST: Attention is the new luxury. GUEST: The product is protecting focus. HOST: That changes how teams work."
	assets := buildPodcastSEOAssets(transcript, "deep work", []string{"focus", "productivity"})
	if len(assets.Titles) != 3 {
		t.Fatalf("titles = %d", len(assets.Titles))
	}
	if !strings.Contains(assets.Notes, "focus, productivity") {
		t.Fatalf("notes missing keywords:\n%s", assets.Notes)
	}
	if len(assets.Quotes) == 0 {
		t.Fatal("expected pull quotes")
	}
}

func TestBlocksToSRT(t *testing.T) {
	srt := blocksToSRT([]string{"HOST: First.", "GUEST: Second."})
	for _, want := range []string{"1\n00:00:00,000 --> 00:00:30,000", "2\n00:00:30,000 --> 00:01:00,000"} {
		if !strings.Contains(srt, want) {
			t.Fatalf("srt missing %q:\n%s", want, srt)
		}
	}
}
