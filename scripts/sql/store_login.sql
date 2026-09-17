-- Store login: client flag, tbl_StoreMaster, and tbl_Leads.StoreMasterID.
-- Target schema: MedLyfe (Database/med-prod-DB.sql). For legacy MediAdmin, replace MedLyfe. with MediAdmin.
-- Run in a transaction in production; verify backup before applying.
-- tbl_Leads.StoreID (varchar) is unchanged — it is a separate free-text field.

BEGIN TRANSACTION;

-- 1) Client master: store-login flag (default off = current single-login behaviour)
IF NOT EXISTS (
    SELECT 1 FROM sys.columns
    WHERE object_id = OBJECT_ID(N'MedLyfe.tbl_ClientMaster') AND name = N'IsStoreLoginEnabled'
)
BEGIN
    ALTER TABLE MedLyfe.tbl_ClientMaster
    ADD IsStoreLoginEnabled BIT NOT NULL
        CONSTRAINT DF_tbl_ClientMaster_IsStoreLoginEnabled DEFAULT (0);
END;

-- 2) Store master (identity band 7000001 avoids tbl_Login.UserID collisions)
IF OBJECT_ID(N'MedLyfe.tbl_StoreMaster', N'U') IS NULL
BEGIN
    CREATE TABLE MedLyfe.tbl_StoreMaster (
        StoreID                     BIGINT IDENTITY(7000001, 1) NOT NULL,
        ClientID                    BIGINT NOT NULL,
        StoreName                   VARCHAR(150) NOT NULL,
        Address                     VARCHAR(150) NOT NULL,
        CityID                      TINYINT NOT NULL,
        StateID                     TINYINT NOT NULL,
        Pincode                     VARCHAR(6) NOT NULL,
        ContactNumber               VARCHAR(10) NOT NULL,
        EmailID                     VARCHAR(75) NOT NULL,
        IsActive                    BIT NOT NULL,
        CreatedBy                   BIGINT NOT NULL,
        CreatedOn                   DATETIME NOT NULL,
        LastUpdatedBy               BIGINT NOT NULL,
        LastUpdatedOn               DATETIME NOT NULL,
        CONSTRAINT PK_tbl_StoreMaster PRIMARY KEY CLUSTERED (StoreID ASC)
    );

    ALTER TABLE MedLyfe.tbl_StoreMaster
        ADD CONSTRAINT DF_tbl_StoreMaster_IsActive DEFAULT (1) FOR [IsActive];
    ALTER TABLE MedLyfe.tbl_StoreMaster
        ADD CONSTRAINT DF_tbl_StoreMaster_CreatedOn DEFAULT (GETDATE()) FOR [CreatedOn];
    ALTER TABLE MedLyfe.tbl_StoreMaster
        ADD CONSTRAINT DF_tbl_StoreMaster_LastUpdatedOn DEFAULT (GETDATE()) FOR [LastUpdatedOn];

    ALTER TABLE MedLyfe.tbl_StoreMaster WITH CHECK
        ADD CONSTRAINT FK_tbl_StoreMaster_tbl_ClientMaster
        FOREIGN KEY (ClientID) REFERENCES MedLyfe.tbl_ClientMaster (ClientID);

    CREATE NONCLUSTERED INDEX IX_tbl_StoreMaster_ClientID
        ON MedLyfe.tbl_StoreMaster (ClientID);

    CREATE UNIQUE NONCLUSTERED INDEX UQ_tbl_StoreMaster_ContactNumber
        ON MedLyfe.tbl_StoreMaster (ContactNumber);

    CREATE UNIQUE NONCLUSTERED INDEX UQ_tbl_StoreMaster_EmailID
        ON MedLyfe.tbl_StoreMaster (EmailID);
END;

-- 3) Leads: optional StoreMasterID. Existing StoreID varchar column is left as-is.
-- StoreMasterID is NULL for leads that do not belong to a store (clients without store login,
-- or client-level leads). Only store-scoped leads set a value. The FK does not require a value:
-- SQL Server allows NULL on a foreign key; a non-NULL value must exist in tbl_StoreMaster.
IF NOT EXISTS (
    SELECT 1 FROM sys.columns
    WHERE object_id = OBJECT_ID(N'MedLyfe.tbl_Leads') AND name = N'StoreMasterID'
)
BEGIN
    ALTER TABLE MedLyfe.tbl_Leads
    ADD StoreMasterID BIGINT NULL;
END;

-- Keep the column nullable if it was added earlier as NOT NULL.
IF EXISTS (
    SELECT 1 FROM sys.columns
    WHERE object_id = OBJECT_ID(N'MedLyfe.tbl_Leads')
      AND name = N'StoreMasterID'
      AND is_nullable = 0
)
BEGIN
    ALTER TABLE MedLyfe.tbl_Leads
    ALTER COLUMN StoreMasterID BIGINT NULL;
END;

IF NOT EXISTS (
    SELECT 1 FROM sys.foreign_keys
    WHERE name = N'FK_tbl_Leads_tbl_StoreMaster' AND parent_object_id = OBJECT_ID(N'MedLyfe.tbl_Leads')
)
BEGIN
    ALTER TABLE MedLyfe.tbl_Leads WITH CHECK
        ADD CONSTRAINT FK_tbl_Leads_tbl_StoreMaster
        FOREIGN KEY (StoreMasterID) REFERENCES MedLyfe.tbl_StoreMaster (StoreID);
END;

IF NOT EXISTS (
    SELECT 1 FROM sys.indexes
    WHERE name = N'IX_tbl_Leads_StoreMasterID' AND object_id = OBJECT_ID(N'MedLyfe.tbl_Leads')
)
BEGIN
    CREATE NONCLUSTERED INDEX IX_tbl_Leads_StoreMasterID
        ON MedLyfe.tbl_Leads (StoreMasterID);
END;

COMMIT TRANSACTION;
