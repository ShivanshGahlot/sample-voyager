package srcdb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fatih/color"
	log "github.com/sirupsen/logrus"
	"github.com/yugabyte/yb-voyager/yb-voyager/src/datafile"
	"github.com/yugabyte/yb-voyager/yb-voyager/src/utils"
	"golang.org/x/exp/slices"
)

type Oracle struct {
	source *Source

	db *sql.DB
}

func newOracle(s *Source) *Oracle {
	return &Oracle{source: s}
}

func (ora *Oracle) Connect() error {
	db, err := sql.Open("godror", ora.getConnectionUri())
	ora.db = db
	return err
}

func (ora *Oracle) CheckRequiredToolsAreInstalled() {
	checkTools("ora2pg", "sqlplus")
}

func (ora *Oracle) GetTableRowCount(tableName string) int64 {
	var rowCount int64
	query := fmt.Sprintf("select count(*) from %s", tableName)

	log.Infof("Querying row count of table %q", tableName)
	err := ora.db.QueryRow(query).Scan(&rowCount)
	if err != nil {
		utils.ErrExit("Failed to query %q for row count of %q: %s", query, tableName, err)
	}
	log.Infof("Table %q has %v rows.", tableName, rowCount)
	return rowCount
}

func (ora *Oracle) GetTableApproxRowCount(tableProgressMetadata *utils.TableProgressMetadata) int64 {
	var approxRowCount sql.NullInt64 // handles case: value of the row is null, default for int64 is 0
	var query string
	if !tableProgressMetadata.IsPartition {
		query = fmt.Sprintf("SELECT NUM_ROWS FROM ALL_TABLES "+
			"WHERE TABLE_NAME='%s'", tableProgressMetadata.TableName)
	} else {
		query = fmt.Sprintf("SELECT NUM_ROWS FROM ALL_TAB_PARTITIONS "+
			"WHERE TABLE_NAME='%s' AND PARTITION_NAME='%s'", tableProgressMetadata.ParentTable, tableProgressMetadata.TableName)
	}

	log.Infof("Querying '%s' approx row count of table %q", query, tableProgressMetadata.TableName)
	err := ora.db.QueryRow(query).Scan(&approxRowCount)
	if err != nil {
		utils.ErrExit("Failed to query %q for approx row count of %q: %s", query, tableProgressMetadata.TableName, err)
	}

	log.Infof("Table %q has approx %v rows.", tableProgressMetadata.TableName, approxRowCount)
	return approxRowCount.Int64
}

func (ora *Oracle) GetVersion() string {
	var version string
	query := "SELECT BANNER FROM V$VERSION"
	// query sample output: Oracle Database 19c Enterprise Edition Release 19.0.0.0.0 - Production
	err := ora.db.QueryRow(query).Scan(&version)
	if err != nil {
		utils.ErrExit("run query %q on source: %s", query, err)
	}
	return version
}

func (ora *Oracle) GetAllTableNames() []string {
	var tableNames []string
	/* below query will collect all tables under given schema except TEMPORARY tables,
	Index related tables(start with DR$) and materialized view */
	query := fmt.Sprintf(`SELECT table_name
		FROM all_tables
		WHERE owner = '%s' AND TEMPORARY = 'N' AND table_name NOT LIKE 'DR$%%'
		AND table_name NOT LIKE 'AQ$%%' AND
		(owner, table_name) not in (
			SELECT owner, mview_name
			FROM all_mviews
			UNION ALL
			SELECT log_owner, log_table
			FROM all_mview_logs)
		ORDER BY table_name ASC`, ora.source.Schema)
	log.Infof(`query used to GetAllTableNames(): "%s"`, query)

	rows, err := ora.db.Query(query)
	if err != nil {
		utils.ErrExit("error in querying source database for table names: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		if err != nil {
			utils.ErrExit("error in scanning query rows for table names: %v", err)
		}

		tableNames = append(tableNames, tableName)
	}

	log.Infof("Table Name List: %q", tableNames)

	return tableNames
}

func (ora *Oracle) GetAllPartitionNames(tableName string) []string {
	query := fmt.Sprintf("SELECT partition_name FROM all_tab_partitions "+
		"WHERE table_name = '%s' AND table_owner = '%s' ORDER BY partition_name ASC",
		tableName, ora.source.Schema)
	rows, err := ora.db.Query(query)
	if err != nil {
		utils.ErrExit("failed to list partitions of table %q: %v", tableName, err)
	}
	defer rows.Close()

	var partitionNames []string
	for rows.Next() {
		var partitionName string
		err = rows.Scan(&partitionName)
		if err != nil {
			utils.ErrExit("error in scanning query rows: %v", err)
		}
		partitionNames = append(partitionNames, partitionName)
		// TODO: Support subpartition(find subparititions for each partition)
	}
	log.Infof("Partition Names for parent table %q: %q", tableName, partitionNames)
	return partitionNames
}

func (ora *Oracle) getConnectionUri() string {
	source := ora.source
	if source.Uri != "" {
		return source.Uri
	}

	switch true {
	case source.DBSid != "":
		source.Uri = fmt.Sprintf(`user="%s" password="%s" connectString="(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=%s)(PORT=%d))(CONNECT_DATA=(SID=%s)))"`,
			source.User, source.Password, source.Host, source.Port, source.DBSid)

	case source.TNSAlias != "":
		source.Uri = fmt.Sprintf(`user="%s" password="%s" connectString="%s"`, source.User, source.Password, source.TNSAlias)

	case source.DBName != "":
		source.Uri = fmt.Sprintf(`user="%s" password="%s" connectString="(DESCRIPTION=(ADDRESS=(PROTOCOL=TCP)(HOST=%s)(PORT=%d))(CONNECT_DATA=(SERVICE_NAME=%s)))"`,
			source.User, source.Password, source.Host, source.Port, source.DBName)
	}

	return source.Uri
}

func (ora *Oracle) ExportSchema(exportDir string) {
	ora2pgExtractSchema(ora.source, exportDir)
}

func (ora *Oracle) ExportData(ctx context.Context, exportDir string, tableList []string, quitChan chan bool, exportDataStart, exportSuccessChan chan bool) {
	ora2pgExportDataOffline(ctx, ora.source, exportDir, tableList, quitChan, exportDataStart, exportSuccessChan)
}

func (ora *Oracle) ExportDataPostProcessing(exportDir string, tablesProgressMetadata map[string]*utils.TableProgressMetadata) {
	exportedRowCount := getExportedRowCount(tablesProgressMetadata)
	dfd := datafile.Descriptor{
		FileFormat:    datafile.SQL,
		TableRowCount: exportedRowCount,
		Delimiter:     "\t",
		HasHeader:     false,
		ExportDir:     exportDir,
	}
	dfd.Save()
}

func (ora *Oracle) GetCharset() (string, error) {
	var charset string
	query := "SELECT VALUE FROM NLS_DATABASE_PARAMETERS WHERE PARAMETER = 'NLS_CHARACTERSET'"
	err := ora.db.QueryRow(query).Scan(&charset)
	if err != nil {
		return "", fmt.Errorf("failed to query %q for database encoding: %s", query, err)
	}
	return charset, nil
}

func (ora *Oracle) FilterUnsupportedTables(tableList []string) []string {
	var unsupportedTables []string

	// query to find unsupported tables
	query := fmt.Sprintf("SELECT queue_table from ALL_QUEUE_TABLES WHERE OWNER = '%s'", ora.source.Schema)
	log.Infof("query for queue tables: %q\n", query)
	rows, err := ora.db.Query(query)
	if err != nil {
		utils.ErrExit("failed to query %q for filtering unsupported queue tables: %v", query, err)
	}
	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		if err != nil {
			utils.ErrExit("failed to scan tableName from output of query %q: %v", query, err)
		}

		if slices.Contains(tableList, tableName) {
			unsupportedTables = append(unsupportedTables, tableName)
		}
	}

	if len(unsupportedTables) > 0 {
		msg := fmt.Sprintf("Ignorning unsupported queue tables: %s\n", unsupportedTables)
		color.Yellow(msg)
		log.Infof(msg)
		tableList = utils.SetDifference(tableList, unsupportedTables)

		fmt.Printf("final table list for data export: %s", tableList)
	}
	return tableList
}
