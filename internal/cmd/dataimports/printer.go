package dataimports

import (
	"github.com/planetscale/cli/internal/printer"
	ps "github.com/planetscale/planetscale-go/planetscale"
	"strings"
)

func PrintDataImport(p *printer.Printer, di ps.DataImport) {
	completedSteps := GetCompletedImportStates(p, di.ImportState)
	if len(completedSteps) > 0 {
		p.Println(completedSteps)
	}

	inProgressStep, _ := GetCurrentImportState(di.ImportState)
	if len(inProgressStep) > 0 {
		p.Println(inProgressStep)
	}

	pendingSteps := GetPendingImportStates(di.ImportState)
	if len(pendingSteps) > 0 {
		p.Println(pendingSteps)
	}
}

func GetCompletedImportStates(p *printer.Printer, state ps.DataImportState) string {
	completedStates := []string{}
	switch state {
	case ps.DataImportCopyingData:
		completedStates = append(completedStates, printer.BoldGreen("1. Started Data Copy"))
	case ps.DataImportSwitchTrafficPending, ps.DataImportSwitchTrafficError:
		completedStates = append(completedStates, printer.BoldGreen("1. Started Data Copy"))
		completedStates = append(completedStates, printer.BoldGreen("2. Copied Data"))
	case ps.DataImportSwitchTrafficCompleted:
		completedStates = append(completedStates, printer.BoldGreen("1. Started Data Copy"))
		completedStates = append(completedStates, printer.BoldGreen("2. Copied Data"))
		completedStates = append(completedStates, printer.BoldGreen("3. Running as Replica"))

	case ps.DataImportReady:
		completedStates = append(completedStates, printer.BoldGreen("1. Started Data Copy"))
		completedStates = append(completedStates, printer.BoldGreen("2. Copied Data"))
		completedStates = append(completedStates, printer.BoldGreen("3. Running as replica"))
		completedStates = append(completedStates, printer.BoldGreen("4. Running as Primary"))
	}
	return strings.Join(completedStates, "\n")
}

func GetCurrentImportState(d ps.DataImportState) (string, bool) {
	switch d {
	case ps.DataImportPreparingDataCopy:
		return printer.BoldYellow("> 1. Starting Data Copy"), true
	case ps.DataImportPreparingDataCopyFailed:
		return printer.BoldRed("> 1. Cannot Start Data Copy"), false
	case ps.DataImportCopyingData:
		return printer.BoldYellow("> 2. Copying Data"), true
	case ps.DataImportCopyingDataFailed:
		return printer.BoldRed("> 2. Failed to Copy Data"), false
	case ps.DataImportSwitchTrafficPending:
		return printer.BoldYellow("> 3. Running as Replica"), true
	case ps.DataImportSwitchTrafficRunning:
		return printer.BoldYellow("> 3. Switching to Primary"), true
	case ps.DataImportSwitchTrafficError:
		return printer.BoldRed("> 3. Failed switching to Primary"), false
	case ps.DataImportReverseTrafficRunning:
		return printer.BoldYellow("3. Switching to replica"), true
	case ps.DataImportSwitchTrafficCompleted:
		return printer.BoldYellow("> 4. Running as Primary"), true
	case ps.DataImportReverseTrafficError:
		return printer.BoldRed("> 3. Failed switching to Primary"), false
	case ps.DataImportDetachExternalDatabaseRunning:
		return printer.BoldYellow("> 4. detaching external database"), true
	case ps.DataImportDetachExternalDatabaseError:
		return printer.BoldRed("> 4. failed to detach external database"), false
	case ps.DataImportReady:
		return printer.BoldGreen("> 5. Ready"), false
	}

	panic("unhandled state " + d.String())
}

func GetPendingImportStates(state ps.DataImportState) string {
	var pendingStates []string
	switch state {
	case ps.DataImportPreparingDataCopy:
		pendingStates = append(pendingStates, printer.BoldBlack("2. Copied Data"))
		pendingStates = append(pendingStates, printer.BoldBlack("3. Running as Replica"))
		pendingStates = append(pendingStates, printer.BoldBlack("4. Running as Primary"))
		pendingStates = append(pendingStates, printer.BoldBlack("5. Detached external database"))
	case ps.DataImportCopyingData:
		pendingStates = append(pendingStates, printer.BoldBlack("3. Running as Replica"))
		pendingStates = append(pendingStates, printer.BoldBlack("4. Running as Primary"))
		pendingStates = append(pendingStates, printer.BoldBlack("5. Detached external database"))
	case ps.DataImportSwitchTrafficPending:
		pendingStates = append(pendingStates, printer.BoldBlack("4. Running as Primary"))
		pendingStates = append(pendingStates, printer.BoldBlack("5. Detached external database"))
	case ps.DataImportSwitchTrafficCompleted:
		pendingStates = append(pendingStates, printer.BoldBlack("5. Detached external database"))
	}
	return strings.Join(pendingStates, "\n")
}

//
//func ImportProgress(state ps.DataImportState) string {
//	preparingDataCopyState := color.New(color.FgBlack).Add(color.Bold).Sprint("1. Started Data Copy")
//	dataCopyState := color.New(color.FgBlack).Add(color.Bold).Sprint("2. Copied Data")
//	switchingTrafficState := color.New(color.FgBlack).Add(color.Bold).Sprint("3. Running as Replica")
//	detachExternalDatabaseState := color.New(color.FgBlack).Add(color.Bold).Sprint("4. Running as Primary")
//	readyState := color.New(color.FgBlack).Add(color.Bold).Sprint("5. Detached External Database, ready to use")
//	// Preparing data copy > Data copying > Running as Replica > Running as Primary > Detach External database
//	switch state {
//	case ps.DataImportPreparingDataCopy:
//		preparingDataCopyState = color.New(color.FgYellow).Add(color.Bold).Sprint(state.String())
//	case ps.DataImportSwitchTrafficPending:
//		preparingDataCopyState = color.New(color.FgGreen).Add(color.Bold).Sprint("1. Started Data Copy")
//		dataCopyState = color.New(color.FgGreen).Add(color.Bold).Sprint("2. Copied Data")
//		switchingTrafficState = color.New(color.FgYellow).Add(color.Bold).Sprint("3. Running as Replica")
//	case ps.DataImportPreparingDataCopyFailed:
//		preparingDataCopyState = color.New(color.FgGreen).Add(color.Bold).Sprint("1. Cannot start data copy")
//	}
//
//	return strings.Join([]string{preparingDataCopyState, dataCopyState, switchingTrafficState, detachExternalDatabaseState, readyState}, "\n")
//
//}
