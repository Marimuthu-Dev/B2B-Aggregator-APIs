-- Lead report approval: tri-state IsFit, download flag, remarks, audit timestamp.
-- Target: MediAdmin.tbl_Leads (SQL Server)
-- Run in a transaction in production; verify backup before applying.

BEGIN TRANSACTION;

-- Step 0: Audit column for approval updates (referenced by application UPDATE)
IF NOT EXISTS (
    SELECT 1 FROM sys.columns
    WHERE object_id = OBJECT_ID(N'MediAdmin.tbl_Leads') AND name = N'FitUpdatedOn'
)
BEGIN
    ALTER TABLE MediAdmin.tbl_Leads
    ADD FitUpdatedOn DATETIME2(3) NULL;
END;

-- Step 1: BIT -> TINYINT (FALSE becomes 0, TRUE becomes 1)
ALTER TABLE MediAdmin.tbl_Leads
ALTER COLUMN IsFit TINYINT NOT NULL;

-- Step 2: Map legacy semantics: TRUE(1)->1 FIT, FALSE(0)->-1 UNFIT
UPDATE MediAdmin.tbl_Leads
SET IsFit = CASE
    WHEN IsFit = 1 THEN 1
    ELSE -1
END;

-- Step 3: Report downloadable even when on hold (or other states when explicitly allowed)
IF NOT EXISTS (
    SELECT 1 FROM sys.columns
    WHERE object_id = OBJECT_ID(N'MediAdmin.tbl_Leads') AND name = N'IsReportDownloadable'
)
BEGIN
    ALTER TABLE MediAdmin.tbl_Leads
    ADD IsReportDownloadable BIT NOT NULL CONSTRAINT DF_tbl_Leads_IsReportDownloadable DEFAULT (0);
END;

-- Step 4: Optional reviewer remarks
IF NOT EXISTS (
    SELECT 1 FROM sys.columns
    WHERE object_id = OBJECT_ID(N'MediAdmin.tbl_Leads') AND name = N'ApprovalRemarks'
)
BEGIN
    ALTER TABLE MediAdmin.tbl_Leads
    ADD ApprovalRemarks VARCHAR(250) NULL;
END;

-- Optional: if you relied on pre-change behavior (any lead with a report could download),
-- grant download for rows that are not FIT, e.g.:
-- UPDATE MediAdmin.tbl_Leads SET IsReportDownloadable = 1 WHERE IsFit <> 1 AND ReportURL IS NOT NULL AND LTRIM(RTRIM(ReportURL)) <> '';

COMMIT TRANSACTION;
