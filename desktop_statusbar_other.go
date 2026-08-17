//go:build desktop && !darwin

package main

func startDesktopStatusBar(*desktopApp) {}

func updateDesktopStatusBar(desktopCodeSummary) {}

func stopDesktopStatusBar() {}
