package constant

import (
	"errors"
)

const (
	CodeSuccess           = 200
	CodeErrBadRequest     = 400
	CodeErrUnauthorized   = 401
	CodeErrNotFound       = 404
	CodeAuth              = 406
	CodeGlobalLoading     = 407
	CodeErrInternalServer = 500

	CodeErrIP           = 310
	CodeErrDomain       = 311
	CodeErrEntrance     = 312
	CodePasswordExpired = 313

	CodeErrXpack = 410

	ErrNotFound       = "ErrNotFound"
	ErrParameterError = "ErrParameterError"
	ErrIdRequired     = "ErrIdRequired"
	ErrRecordNotFound = "ErrRecordNotFound"
)

// internal
var (
	ErrCaptchaCode     = errors.New("ErrCaptchaCode")
	ErrAuth            = errors.New("ErrAuth")
	ErrRecordExist     = errors.New("ErrRecordExist")
	ErrRecordNotExist  = errors.New("ErrRecordNotExist")
	ErrStructTransform = errors.New("ErrStructTransform")
	ErrInitialPassword = errors.New("ErrInitialPassword")
	ErrNotSupportType  = errors.New("ErrNotSupportType")
	ErrInvalidParams   = errors.New("ErrInvalidParams")

	ErrTokenParse = errors.New("ErrTokenParse")
)

// api
var (
	ErrTypeInternalServer      = "ErrInternalServer"
	ErrTypeInvalidParams       = "ErrInvalidParams"
	ErrTypeNotLogin            = "ErrNotLogin"
	ErrTypePasswordExpired     = "ErrPasswordExpired"
	ErrNameIsExist             = "ErrNameIsExist"
	ErrDemoEnvironment         = "ErrDemoEnvironment"
	ErrCmdIllegal              = "ErrCmdIllegal"
	ErrXpackNotFound           = "ErrXpackNotFound"
	ErrXpackNotActive          = "ErrXpackNotActive"
	ErrXpackLost               = "ErrXpackLost"
	ErrXpackTimeout            = "ErrXpackTimeout"
	ErrXpackOutOfDate          = "ErrXpackOutOfDate"
	ErrApiConfigStatusInvalid  = "ErrApiConfigStatusInvalid"
	ErrApiConfigKeyInvalid     = "ErrApiConfigKeyInvalid"
	ErrApiConfigIPInvalid      = "ErrApiConfigIPInvalid"
	ErrApiConfigDisable        = "ErrApiConfigDisable"
	ErrApiConfigKeyTimeInvalid = "ErrApiConfigKeyTimeInvalid"
)

// app
var (
	ErrPortInUsed                    = "ErrPortInUsed"
	ErrAppLimit                      = "ErrAppLimit"
	ErrNotInstall                    = "ErrNotInstall"
	ErrPortInOtherApp                = "ErrPortInOtherApp"
	ErrDbUserNotValid                = "ErrDbUserNotValid"
	ErrDatabaseUserIdentityRequired  = "ErrDatabaseUserIdentityRequired"
	ErrDatabaseUserHostRequired      = "ErrDatabaseUserHostRequired"
	ErrUpdateBuWebsite               = "ErrUpdateBuWebsite"
	ErrGoPanelNetworkFailed          = "ErrGoPanelNetworkFailed"
	ErrCmdTimeout                    = "ErrCmdTimeout"
	ErrFileParse                     = "ErrFileParse"
	ErrInstallDirNotFound            = "ErrInstallDirNotFound"
	ErrContainerName                 = "ErrContainerName"
	ErrAppNameExist                  = "ErrAppNameExist"
	ErrFileNotFound                  = "ErrFileNotFound"
	ErrFileParseApp                  = "ErrFileParseApp"
	ErrAppParamKey                   = "ErrAppParamKey"
	ErrPipelineExpectedCommitInvalid = "ErrPipelineExpectedCommitInvalid"
	ErrPipelineExpectedCommitRepo    = "ErrPipelineExpectedCommitRepo"
	ErrPipelineSourceInvalid         = "ErrPipelineSourceInvalid"
	ErrPipelineCodeProjectRequired   = "ErrPipelineCodeProjectRequired"
	ErrPipelineCodeProjectNotFound   = "ErrPipelineCodeProjectNotFound"
	ErrPipelineForceDeleteName       = "ErrPipelineForceDeleteName"
	ErrPipelineForceDeleteRunning    = "ErrPipelineForceDeleteRunning"
	ErrPipelineForceDeleteFlow       = "ErrPipelineForceDeleteFlow"
	ErrPipelineForceDeleteHistory    = "ErrPipelineForceDeleteHistory"
	ErrPipelineForceDeleteWebsite    = "ErrPipelineForceDeleteWebsite"
	ErrFlowNameRequired              = "ErrFlowNameRequired"
	ErrFlowProjectRequired           = "ErrFlowProjectRequired"
	ErrFlowPipelineRequired          = "ErrFlowPipelineRequired"
	ErrFlowEnvironmentRequired       = "ErrFlowEnvironmentRequired"
	ErrFlowProjectNotFound           = "ErrFlowProjectNotFound"
	ErrFlowPipelineNotFound          = "ErrFlowPipelineNotFound"
	ErrFlowPipelineProjectMismatch   = "ErrFlowPipelineProjectMismatch"
	ErrFlowWebsiteNotFound           = "ErrFlowWebsiteNotFound"
	ErrFlowProjectForbidden          = "ErrFlowProjectForbidden"
	ErrFlowProjectExists             = "ErrFlowProjectExists"
	ErrFlowEnvironmentInvalid        = "ErrFlowEnvironmentInvalid"
	ErrFlowNotFound                  = "ErrFlowNotFound"
	ErrFlowForbidden                 = "ErrFlowForbidden"
	ErrFlowDeleteHistory             = "ErrFlowDeleteHistory"
	ErrFlowDisabled                  = "ErrFlowDisabled"
	ErrFlowCommitRequired            = "ErrFlowCommitRequired"
	ErrFlowCodeDeliveryRequired      = "ErrFlowCodeDeliveryRequired"
	ErrFlowCodeDeliveryInvalid       = "ErrFlowCodeDeliveryInvalid"
	ErrFlowCodeBaselineInvalid       = "ErrFlowCodeBaselineInvalid"
	ErrFlowCodeBaselineUnavailable   = "ErrFlowCodeBaselineUnavailable"
	ErrFlowCodeSourceInvalid         = "ErrFlowCodeSourceInvalid"
	ErrFlowVersionInvalid            = "ErrFlowVersionInvalid"
	ErrFlowVersionExists             = "ErrFlowVersionExists"
	ErrFlowRunNotFailed              = "ErrFlowRunNotFailed"
	ErrFlowRunResumeUnsupported      = "ErrFlowRunResumeUnsupported"
	ErrFlowRunRebuildUnsupported     = "ErrFlowRunRebuildUnsupported"
	ErrFlowPipelineRecordProtected   = "ErrFlowPipelineRecordProtected"
)

// website
var (
	ErrDomainIsExist      = "ErrDomainIsExist"
	ErrAliasIsExist       = "ErrAliasIsExist"
	ErrGroupIsUsed        = "ErrGroupIsUsed"
	ErrUsernameIsExist    = "ErrUsernameIsExist"
	ErrUsernameIsNotExist = "ErrUsernameIsNotExist"
	ErrBackupMatch        = "ErrBackupMatch"
	ErrBackupExist        = "ErrBackupExist"
	ErrDomainIsUsed       = "ErrDomainIsUsed"
)

// ssl
var (
	ErrSSLCannotDelete               = "ErrSSLCannotDelete"
	ErrAccountCannotDelete           = "ErrAccountCannotDelete"
	ErrSSLApply                      = "ErrSSLApply"
	ErrEmailIsExist                  = "ErrEmailIsExist"
	ErrEabKidOrEabHmacKeyCannotBlank = "ErrEabKidOrEabHmacKeyCannotBlank"
)

// file
var (
	ErrPathNotFound     = "ErrPathNotFound"
	ErrMovePathFailed   = "ErrMovePathFailed"
	ErrLinkPathNotFound = "ErrLinkPathNotFound"
	ErrFileIsExist      = "ErrFileIsExist"
	ErrFileUpload       = "ErrFileUpload"
	ErrFileDownloadDir  = "ErrFileDownloadDir"
	ErrCmdNotFound      = "ErrCmdNotFound"
	ErrFavoriteExist    = "ErrFavoriteExist"
	ErrPathNotDelete    = "ErrPathNotDelete"
)

// mysql
var (
	ErrUserIsExist     = "ErrUserIsExist"
	ErrDatabaseIsExist = "ErrDatabaseIsExist"
	ErrExecTimeOut     = "ErrExecTimeOut"
	ErrRemoteExist     = "ErrRemoteExist"
	ErrLocalExist      = "ErrLocalExist"
)

// redis
var (
	ErrTypeOfRedis = "ErrTypeOfRedis"
)

// container
var (
	ErrInUsed            = "ErrInUsed"
	ErrObjectInUsed      = "ErrObjectInUsed"
	ErrObjectBeDependent = "ErrObjectBeDependent"
	ErrPortRules         = "ErrPortRules"
	ErrPgImagePull       = "ErrPgImagePull"
)

// runtime
var (
	ErrDirNotFound         = "ErrDirNotFound"
	ErrFileNotExist        = "ErrFileNotExist"
	ErrImageBuildErr       = "ErrImageBuildErr"
	ErrImageExist          = "ErrImageExist"
	ErrDelWithWebsite      = "ErrDelWithWebsite"
	ErrRuntimeStart        = "ErrRuntimeStart"
	ErrPackageJsonNotFound = "ErrPackageJsonNotFound"
	ErrScriptsNotFound     = "ErrScriptsNotFound"
)

var (
	ErrBackupInUsed = "ErrBackupInUsed"
	ErrOSSConn      = "ErrOSSConn"
	ErrEntrance     = "ErrEntrance"
)

var (
	ErrFirewallNone = "ErrFirewallNone"
	ErrFirewallBoth = "ErrFirewallBoth"
)

// cronjob
var (
	ErrBashExecute = "ErrBashExecute"
)

var (
	ErrNotExistUser = "ErrNotExistUser"
)

// license
var (
	ErrLicense      = "ErrLicense"
	ErrLicenseCheck = "ErrLicenseCheck"
	ErrXpackVersion = "ErrXpackVersion"
	ErrLicenseSave  = "ErrLicenseSave"
	ErrLicenseSync  = "ErrLicenseSync"
)

// alert
var (
	ErrAlert       = "ErrAlert"
	ErrAlertPush   = "ErrAlertPush"
	ErrAlertSave   = "ErrAlertSave"
	ErrAlertSync   = "ErrAlertSync"
	ErrAlertRemote = "ErrAlertRemote"
)

// mobile app
var (
	ErrVerifyToken  = "ErrVerifyToken"
	ErrInvalidToken = "ErrInvalidToken"
	ErrExpiredToken = "ErrExpiredToken"
)

// auth / user validation
var (
	ErrCaptchaRequired     = "ErrCaptchaRequired"
	ErrLoginIPBlocked      = "ErrLoginIPBlocked"
	ErrLoginAccountBlocked = "ErrLoginAccountBlocked"
	ErrTokenRequired       = "ErrTokenRequired"
	ErrEmailRequired       = "ErrEmailRequired"
	ErrPasswordRequired    = "ErrPasswordRequired"
	ErrUserNoteTooLong     = "ErrUserNoteTooLong"
)

// host / disk / terminal
var (
	ErrHostDiskScanTaskMissing             = "ErrHostDiskScanTaskMissing"
	ErrHostDiskNoFileSelected              = "ErrHostDiskNoFileSelected"
	ErrHostDiskTaskIDRequired              = "ErrHostDiskTaskIDRequired"
	ErrHostCPURelieveMessage               = "ErrHostCPURelieveMessage"
	ErrHostTerminalSessionEnded            = "ErrHostTerminalSessionEnded"
	ErrHostTerminalProcessGone             = "ErrHostTerminalProcessGone"
	ErrHostTerminalSessionEndedResume      = "ErrHostTerminalSessionEndedResume"
	ErrHostTerminalRunningForbid           = "ErrHostTerminalRunningForbid"
	ErrHostTerminalSessionNotFound         = "ErrHostTerminalSessionNotFound"
	ErrHostTerminalSessionIDInvalid        = "ErrHostTerminalSessionIDInvalid"
	ErrHostTerminalProcessEnded            = "ErrHostTerminalProcessEnded"
	ErrHostTerminalWorkDirInvalid          = "ErrHostTerminalWorkDirInvalid"
	ErrHostTerminalWorkDirInaccessible     = "ErrHostTerminalWorkDirInaccessible"
	ErrHostTerminalSubscriberMissing       = "ErrHostTerminalSubscriberMissing"
	ErrHostTerminalControlledByOther       = "ErrHostTerminalControlledByOther"
	ErrHostTerminalNoInputRight            = "ErrHostTerminalNoInputRight"
	ErrHostTerminalNoControlRight          = "ErrHostTerminalNoControlRight"
	ErrHostTerminalShellUnsupported        = "ErrHostTerminalShellUnsupported"
	ErrHostTerminalShellNotInstalled       = "ErrHostTerminalShellNotInstalled"
	ErrHostTerminalProcessMissing          = "ErrHostTerminalProcessMissing"
	ErrHostTerminalPowerShellMissing       = "ErrHostTerminalPowerShellMissing"
	ErrHostTerminalCmdUnavailable          = "ErrHostTerminalCmdUnavailable"
	ErrHostTerminalAuditStop               = "ErrHostTerminalAuditStop"
	ErrHostTerminalAuditReconnect          = "ErrHostTerminalAuditReconnect"
	ErrHostTerminalAuditDelete             = "ErrHostTerminalAuditDelete"
	ErrHostTerminalInterruptedMessage      = "ErrHostTerminalInterruptedMessage"
	ErrHostTerminalReadOnly                = "ErrHostTerminalReadOnly"
	ErrHostTerminalWritable                = "ErrHostTerminalWritable"
	ErrHostMemGoPanelRecycledLinuxNoDrop   = "ErrHostMemGoPanelRecycledLinuxNoDrop"
	ErrHostMemRootRequiredLinux            = "ErrHostMemRootRequiredLinux"
	ErrHostMemCacheCleanedLinux            = "ErrHostMemCacheCleanedLinux"
	ErrHostMemGoPanelRecycledDarwinNoPurge = "ErrHostMemGoPanelRecycledDarwinNoPurge"
	ErrHostMemRootRequiredDarwin           = "ErrHostMemRootRequiredDarwin"
	ErrHostMemCacheCleanedDarwin           = "ErrHostMemCacheCleanedDarwin"
	ErrHostMemGoPanelRecycledWindows       = "ErrHostMemGoPanelRecycledWindows"
	ErrHostMemGoPanelRecycledUnsupported   = "ErrHostMemGoPanelRecycledUnsupported"
)

// container / podman / docker / website binding
var (
	ErrContainerUnsupportedPlatform               = "ErrContainerUnsupportedPlatform"
	ErrContainerGpcOutdatedPodmanSocket           = "ErrContainerGpcOutdatedPodmanSocket"
	ErrContainerGpcOutdatedLinger                 = "ErrContainerGpcOutdatedLinger"
	ErrContainerWebsiteBindEmpty                  = "ErrContainerWebsiteBindEmpty"
	ErrContainerWebsiteRuntimeLoadFailed          = "ErrContainerWebsiteRuntimeLoadFailed"
	ErrContainerWebsiteNotFound                   = "ErrContainerWebsiteNotFound"
	ErrContainerWebsiteNotRunning                 = "ErrContainerWebsiteNotRunning"
	ErrContainerWebsiteStale                      = "ErrContainerWebsiteStale"
	ErrContainerWebsitePortMismatch               = "ErrContainerWebsitePortMismatch"
	ErrContainerWebsiteUpstreamUnreachable        = "ErrContainerWebsiteUpstreamUnreachable"
	ErrContainerWebsiteUpstreamBadStatus          = "ErrContainerWebsiteUpstreamBadStatus"
	ErrContainerWebsiteMissing                    = "ErrContainerWebsiteMissing"
	ErrContainerWebsiteNotReverseProxy            = "ErrContainerWebsiteNotReverseProxy"
	ErrContainerWebsiteAppStoreForbidden          = "ErrContainerWebsiteAppStoreForbidden"
	ErrContainerWebsiteApplyFailed                = "ErrContainerWebsiteApplyFailed"
	ErrContainerPodmanAutoStartFailed             = "ErrContainerPodmanAutoStartFailed"
	ErrContainerPodmanNoSocket                    = "ErrContainerPodmanNoSocket"
	ErrContainerPodmanDarwinNoLogClean            = "ErrContainerPodmanDarwinNoLogClean"
	ErrContainerLogPathEmpty                      = "ErrContainerLogPathEmpty"
	ErrContainerRuntimeInstallIDRequired          = "ErrContainerRuntimeInstallIDRequired"
	ErrContainerRuntimeInstallRuntimeRequired     = "ErrContainerRuntimeInstallRuntimeRequired"
	ErrContainerRuntimeInstallUnsupportedPlatform = "ErrContainerRuntimeInstallUnsupportedPlatform"
	ErrContainerRuntimeInstallAlreadyInstalled    = "ErrContainerRuntimeInstallAlreadyInstalled"
	ErrContainerRuntimeInstallInProgress          = "ErrContainerRuntimeInstallInProgress"
	ErrContainerRuntimeInstallTaskNotFound        = "ErrContainerRuntimeInstallTaskNotFound"
	ErrContainerRuntimeInstallGpcOutdated         = "ErrContainerRuntimeInstallGpcOutdated"
	ErrContainerJournalUserPermission             = "ErrContainerJournalUserPermission"
	ErrContainerJournalPermission                 = "ErrContainerJournalPermission"
	ErrContainerPodmanSocketStillUnavailable      = "ErrContainerPodmanSocketStillUnavailable"
	ErrContainerPodmanSocketLingerHint            = "ErrContainerPodmanSocketLingerHint"
)

// website
var (
	ErrWebsiteNotFound                  = "ErrWebsiteNotFound"
	ErrWebsitePipelineDeprecated        = "ErrWebsitePipelineDeprecated"
	ErrWebsitePathAliasExist            = "ErrWebsitePathAliasExist"
	ErrWebsiteDomainPrimaryRequired     = "ErrWebsiteDomainPrimaryRequired"
	ErrWebsiteDomainAtLeastOne          = "ErrWebsiteDomainAtLeastOne"
	ErrWebsiteDomainHTTPPrefix          = "ErrWebsiteDomainHTTPPrefix"
	ErrWebsiteUpstreamAtLeastOne        = "ErrWebsiteUpstreamAtLeastOne"
	ErrWebsiteUpstreamAtLeastOneEnabled = "ErrWebsiteUpstreamAtLeastOneEnabled"
	ErrWebsiteAliasEmpty                = "ErrWebsiteAliasEmpty"
	ErrWebsiteStaticPathEmpty           = "ErrWebsiteStaticPathEmpty"
	ErrWebsiteIDNotFound                = "ErrWebsiteIDNotFound"
	ErrWebsitePrimaryDomainError        = "ErrWebsitePrimaryDomainError"
	ErrWebsiteOtherDomainsError         = "ErrWebsiteOtherDomainsError"
)

// database
var (
	ErrDatabaseGetServerFailed    = "ErrDatabaseGetServerFailed"
	ErrDatabaseCreateUserFailed   = "ErrDatabaseCreateUserFailed"
	ErrDatabaseCreateDBFailed     = "ErrDatabaseCreateDBFailed"
	ErrDatabaseGrantUserFailed    = "ErrDatabaseGrantUserFailed"
	ErrDatabaseSQLiteInaccessible = "ErrDatabaseSQLiteInaccessible"
	ErrDatabaseInsertSuccess      = "ErrDatabaseInsertSuccess"
	ErrDatabaseUpdateSuccess      = "ErrDatabaseUpdateSuccess"
	ErrDatabaseDeleteSuccess      = "ErrDatabaseDeleteSuccess"
	ErrDatabaseCreateSuccess      = "ErrDatabaseCreateSuccess"
	ErrDatabaseCopySuccess        = "ErrDatabaseCopySuccess"
	ErrDatabaseModifySuccess      = "ErrDatabaseModifySuccess"
)

// file
var (
	ErrFileAccessDenied          = "ErrFileAccessDenied"
	ErrFileSubAdminNoBaseDir     = "ErrFileSubAdminNoBaseDir"
	ErrFileKeyRequired           = "ErrFileKeyRequired"
	ErrFileUnauthorized          = "ErrFileUnauthorized"
	ErrFileInvalidTaskKey        = "ErrFileInvalidTaskKey"
	ErrFilePermissionDenied      = "ErrFilePermissionDenied"
	ErrFileCompressTaskSubmitted = "ErrFileCompressTaskSubmitted"
	ErrFileCompressStart         = "ErrFileCompressStart"
	ErrFileCompressFileCount     = "ErrFileCompressFileCount"
	ErrFileCompressFailed        = "ErrFileCompressFailed"
	ErrFileCompressCompleted     = "ErrFileCompressCompleted"
	ErrFileWgetStart             = "ErrFileWgetStart"
	ErrFileWgetSavePath          = "ErrFileWgetSavePath"
	ErrFileWgetSSLIgnored        = "ErrFileWgetSSLIgnored"
	ErrFileWgetCancelled         = "ErrFileWgetCancelled"
	ErrFileWgetFailed            = "ErrFileWgetFailed"
	ErrFileWgetCompleted         = "ErrFileWgetCompleted"
	ErrFileWgetTaskSubmitted     = "ErrFileWgetTaskSubmitted"
)

// apps / app / install / repair
var (
	ErrAppsKeyRequired                = "ErrAppsKeyRequired"
	ErrAppsInvalidID                  = "ErrAppsInvalidID"
	ErrAppsUnsupportedPlatform        = "ErrAppsUnsupportedPlatform"
	ErrAppsGpcOutdatedComposeInstall  = "ErrAppsGpcOutdatedComposeInstall"
	ErrAppsGpcOutdatedPodmanSubUID    = "ErrAppsGpcOutdatedPodmanSubUID"
	ErrAppsGpcOutdatedPodmanShortName = "ErrAppsGpcOutdatedPodmanShortName"
	ErrAppsSubUIDRepairUnnecessary    = "ErrAppsSubUIDRepairUnnecessary"
	ErrAppsNoPortConflict             = "ErrAppsNoPortConflict"
	ErrAppsEnvUpdateFailed            = "ErrAppsEnvUpdateFailed"
	ErrAppsPortConflictResolved       = "ErrAppsPortConflictResolved"
	ErrAppsResourceInsufficientTitle  = "ErrAppsResourceInsufficientTitle"
	ErrAppsResourceInsufficientMemory = "ErrAppsResourceInsufficientMemory"
	ErrAppsResourceInsufficientDisk   = "ErrAppsResourceInsufficientDisk"
	ErrAppsResourceInsufficientHint   = "ErrAppsResourceInsufficientHint"
)

// pipeline / flow
var (
	ErrPipelinePermissionDenied        = "ErrPipelinePermissionDenied"
	ErrPipelineInvalidReleaseID        = "ErrPipelineInvalidReleaseID"
	ErrPipelineLogFoldedNote           = "ErrPipelineLogFoldedNote"
	ErrPipelineStreamFoldedNote        = "ErrPipelineStreamFoldedNote"
	ErrFlowDeliveryNoCommit            = "ErrFlowDeliveryNoCommit"
	ErrFlowDeliveryInvalidCommit       = "ErrFlowDeliveryInvalidCommit"
	ErrFlowDeliveryDuplicateMapping    = "ErrFlowDeliveryDuplicateMapping"
	ErrFlowDeliveryBaselineUnavailable = "ErrFlowDeliveryBaselineUnavailable"
	ErrFlowDeliveryOutsideSource       = "ErrFlowDeliveryOutsideSource"
	ErrFlowRuntimeUnavailable          = "ErrFlowRuntimeUnavailable"
	ErrFlowRunnerEmpty                 = "ErrFlowRunnerEmpty"
	ErrFlowProjectNoSource             = "ErrFlowProjectNoSource"
	ErrFlowProjectRepoHeadInvalid      = "ErrFlowProjectRepoHeadInvalid"
	ErrFlowProjectRepoDuplicate        = "ErrFlowProjectRepoDuplicate"
	ErrFlowProjectNoGitRepo            = "ErrFlowProjectNoGitRepo"
	ErrFlowProjectSourceInaccessible   = "ErrFlowProjectSourceInaccessible"
	ErrFlowProjectGitRepoExceeded      = "ErrFlowProjectGitRepoExceeded"
	ErrFlowGitOperationFailed          = "ErrFlowGitOperationFailed"
)

// code / ai workspace
var (
	ErrCodeSessionIDInvalid                 = "ErrCodeSessionIDInvalid"
	ErrCodeInvalidApprovalPolicy            = "ErrCodeInvalidApprovalPolicy"
	ErrCodeApprovalPolicyUnsupported        = "ErrCodeApprovalPolicyUnsupported"
	ErrCodeWorktreeBranchMismatch           = "ErrCodeWorktreeBranchMismatch"
	ErrCodeWorktreeMetadataIncomplete       = "ErrCodeWorktreeMetadataIncomplete"
	ErrCodeProjectSourceDirNotAbs           = "ErrCodeProjectSourceDirNotAbs"
	ErrCodeProjectSourcePathNotDir          = "ErrCodeProjectSourcePathNotDir"
	ErrCodeWorktreeOutsideManaged           = "ErrCodeWorktreeOutsideManaged"
	ErrCodeWorktreeDirIDMismatch            = "ErrCodeWorktreeDirIDMismatch"
	ErrCodeMultiWorktreeOutsideManaged      = "ErrCodeMultiWorktreeOutsideManaged"
	ErrCodeMultiWorktreeDirIDMismatch       = "ErrCodeMultiWorktreeDirIDMismatch"
	ErrCodeMultiWorktreeMetadataUnavailable = "ErrCodeMultiWorktreeMetadataUnavailable"
	ErrCodeRepoWorktreeOutsideManaged       = "ErrCodeRepoWorktreeOutsideManaged"
	ErrCodeSessionDirNotWorktreeRoot        = "ErrCodeSessionDirNotWorktreeRoot"
	ErrCodeSessionSourceDirNotRepoRoot      = "ErrCodeSessionSourceDirNotRepoRoot"
	ErrCodeWorktreeGitCommonDirMismatch     = "ErrCodeWorktreeGitCommonDirMismatch"
	ErrCodeWorktreeGitPrivateDirInvalid     = "ErrCodeWorktreeGitPrivateDirInvalid"
	ErrCodeGitWritableDirOutOfMetadata      = "ErrCodeGitWritableDirOutOfMetadata"
	ErrCodeGitMetadataPathNotAbs            = "ErrCodeGitMetadataPathNotAbs"
	ErrCodeGitMetadataPathNotDir            = "ErrCodeGitMetadataPathNotDir"
)

// cronjob / backup / monitor / logs
var (
	ErrCronjobScriptEmpty              = "ErrCronjobScriptEmpty"
	ErrCronjobDBServerRequired         = "ErrCronjobDBServerRequired"
	ErrCronjobDBTypeAndDBRequired      = "ErrCronjobDBTypeAndDBRequired"
	ErrCronjobLogTypeRequired          = "ErrCronjobLogTypeRequired"
	ErrCronjobUnknownType              = "ErrCronjobUnknownType"
	ErrCronjobSchedulerUninitialized   = "ErrCronjobSchedulerUninitialized"
	ErrCronjobGetDBServerFailed        = "ErrCronjobGetDBServerFailed"
	ErrCronjobUnsupportedDBType        = "ErrCronjobUnsupportedDBType"
	ErrCronjobCleanOpLogFailed         = "ErrCronjobCleanOpLogFailed"
	ErrCronjobCleanLoginLogFailed      = "ErrCronjobCleanLoginLogFailed"
	ErrCronjobCertAutoRenewQueryFailed = "ErrCronjobCertAutoRenewQueryFailed"
	ErrCronjobOpLogCleaned             = "ErrCronjobOpLogCleaned"
	ErrCronjobLoginLogCleaned          = "ErrCronjobLoginLogCleaned"
	ErrCronjobCertRenewFailed          = "ErrCronjobCertRenewFailed"
	ErrCronjobCertRenewed              = "ErrCronjobCertRenewed"
	ErrCronjobNoCertsToRenew           = "ErrCronjobNoCertsToRenew"
	ErrBackupPrepare                   = "ErrBackupPrepare"
	ErrBackupFileSaved                 = "ErrBackupFileSaved"
	ErrBackupRecordSaveFailed          = "ErrBackupRecordSaveFailed"
	ErrBackupRestoreFromUpload         = "ErrBackupRestoreFromUpload"
	ErrLogSshLoginUnsupported          = "ErrLogSshLoginUnsupported"
	ErrMonitorLoadFailed               = "ErrMonitorLoadFailed"
)

// setting / ssl / notify / node
var (
	ErrNodeIDInvalid                = "ErrNodeIDInvalid"
	ErrSettingAPITokenPersistFailed = "ErrSettingAPITokenPersistFailed"
	ErrSettingPortInvalid           = "ErrSettingPortInvalid"
	ErrSettingPortUsedByProcess     = "ErrSettingPortUsedByProcess"
	ErrSettingPortInUse             = "ErrSettingPortInUse"
	ErrSettingEntranceEmpty         = "ErrSettingEntranceEmpty"
	ErrSettingEntranceConflicts     = "ErrSettingEntranceConflicts"
	ErrSettingUpdateInfoUnavailable = "ErrSettingUpdateInfoUnavailable"
	ErrNotifyTestEmailSent          = "ErrNotifyTestEmailSent"
	ErrNotifyAlertRoundExecuted     = "ErrNotifyAlertRoundExecuted"
)

// mobile / daemon / dashboard
var (
	ErrMobilePairingTooFrequent       = "ErrMobilePairingTooFrequent"
	ErrMobileDeviceParamsInvalid      = "ErrMobileDeviceParamsInvalid"
	ErrMobileContainerListFormat      = "ErrMobileContainerListFormat"
	ErrMobileContainerParamsInvalid   = "ErrMobileContainerParamsInvalid"
	ErrMobileContainerActionForbidden = "ErrMobileContainerActionForbidden"
)

// gp-agent / gpc
var (
	ErrAgentCaddyConfigNotFound         = "ErrAgentCaddyConfigNotFound"
	ErrAgentHTTPServicePortOccupied     = "ErrAgentHTTPServicePortOccupied"
	ErrAgentHTTPServicePermissionDenied = "ErrAgentHTTPServicePermissionDenied"
	ErrAgentHTTPServiceStartFailed      = "ErrAgentHTTPServiceStartFailed"
	ErrAgentHTTPServiceStopFailed       = "ErrAgentHTTPServiceStopFailed"
	ErrAgentCaddyAdapterNotFound        = "ErrAgentCaddyAdapterNotFound"
	ErrAgentRegistriesConfExists        = "ErrAgentRegistriesConfExists"
	ErrAgentSSHLogSourceNotFound        = "ErrAgentSSHLogSourceNotFound"
)
