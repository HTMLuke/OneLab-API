package fileService

import "testing"

func TestFilenameFromContentDispositionPrefersRFC5987Filename(t *testing.T) {
	got, err := filenameFromContentDisposition(`attachment; filename="fallback.pdf"; filename*=utf-8''2026-09-30%20R%20A%20Kontoauszug%2009-26.pdf`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if want := "Test.pdf"; got != want {
		t.Fatalf("expected filename %q, got %q", want, got)
	}
}

func TestFilenameFromContentDispositionRequiresFilename(t *testing.T) {
	_, err := filenameFromContentDisposition("attachment")
	if err == nil {
		t.Fatal("expected missing filename to fail")
	}
}

func TestFindPaperlessContainerNameFromDockerPsOutput(t *testing.T) {
	output := "paperless-webserver\tabc123\npaperless-db\tdef456\n"

	got, err := findPaperlessContainerNameFromDockerPsOutput(output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "paperless-webserver" {
		t.Fatalf("expected paperless-webserver, got %q", got)
	}
}

func TestFindPaperlessContainerNameFromDockerPsOutput_UsesFallbackPaperlessName(t *testing.T) {
	output := "some-other\tzzz\npaperless\tqqq\n"

	got, err := findPaperlessContainerNameFromDockerPsOutput(output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "paperless" {
		t.Fatalf("expected paperless, got %q", got)
	}
}
