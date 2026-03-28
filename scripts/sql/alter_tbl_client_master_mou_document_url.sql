-- MediAdmin.tbl_ClientMaster: optional MoU PDF URL (Azure Blob or other HTTPS URL)
-- Run against the target database after backup / change window.

IF NOT EXISTS (
    SELECT 1
    FROM sys.columns
    WHERE object_id = OBJECT_ID(N'[MediAdmin].[tbl_ClientMaster]')
      AND name = N'MoUDocumentURL'
)
BEGIN
    ALTER TABLE [MediAdmin].[tbl_ClientMaster]
    ADD [MoUDocumentURL] VARCHAR(500) NULL;
END
GO
