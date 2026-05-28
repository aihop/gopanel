import { computed, h, onMounted, ref, watch } from 'vue'
import { NButton, NPopconfirm } from 'naive-ui'
import { execDBManagerSqlAPI } from '@/api/modules/database'

type MessageLike = {
  success: (content: string) => void
  error: (content: string) => void
  warning: (content: string) => void
}

type StructureProps = {
  selectedServerId: number | null
  selectedDatabase: string | null
  selectedTable: string | null
  serverOptions: any[]
  structureData: any[]
  loadingStructure: boolean
}

type EmitLike = {
  refresh: () => void
}

export const useStructureView = (
  props: StructureProps,
  emit: EmitLike,
  message: MessageLike
) => {
  const showColumnModal = ref(false)
  const isEditColumn = ref(false)
  const submittingColumn = ref(false)
  const columnForm = ref({
    oldName: '',
    name: '',
    type: '',
    nullable: true,
    defaultValue: '',
    comment: '',
    autoIncrement: false,
    afterColumn: ''
  })

  const indexData = ref<any[]>([])
  const loadingIndex = ref(false)
  const showIndexModal = ref(false)
  const isEditIndex = ref(false)
  const submittingIndex = ref(false)
  const indexForm = ref({
    oldName: '',
    oldType: 'INDEX',
    oldColumns: [] as string[],
    name: '',
    type: 'INDEX',
    columns: [] as string[]
  })

  const indexTypeOptions = [
    { label: 'PRIMARY', value: 'PRIMARY' },
    { label: 'UNIQUE', value: 'UNIQUE' },
    { label: 'INDEX', value: 'INDEX' }
  ]

  const selectedServer = computed(() => {
    return props.serverOptions.find(s => s.value === props.selectedServerId) || null
  })

  const selectedServerLabel = computed(() => {
    return selectedServer.value?.label || ''
  })

  const fieldSummary = computed(() => {
    const rows = Array.isArray(props.structureData) ? props.structureData : []
    const primaryCount = rows.filter((row: any) => row.Key === 'PRI').length
    const nullableCount = rows.filter((row: any) => row.Null === 'YES' || row.Null === 1 || row.Null === true).length
    return {
      total: rows.length,
      primaryCount,
      nullableCount,
      autoIncrementCount: rows.filter((row: any) => String(row.Extra || '').toLowerCase().includes('auto_increment')).length
    }
  })

  const indexSummary = computed(() => {
    const rows = indexData.value || []
    return {
      total: rows.length,
      uniqueCount: rows.filter((row: any) => row.Non_unique === 0).length,
      primaryCount: rows.filter((row: any) => row.Key_name === 'PRIMARY' || String(row.Key_name || '').endsWith('_pkey')).length
    }
  })

  const indexColumnsOptions = computed(() => {
    return props.structureData.map((col: any) => ({
      label: col.Field,
      value: col.Field
    }))
  })

  const afterColumnOptions = computed(() => {
    const rows = Array.isArray(props.structureData) ? props.structureData : []
    const options = rows.map((col: any) => ({
      label: col.Field,
      value: col.Field
    }))
    // Allow placing at the first position
    options.unshift({ label: '(第一列)', value: '__FIRST__' })
    return options
  })

  const normalizeIndexType = (row: any) => {
    const keyName = String(row.Key_name || '')
    const rawType = String(row.Index_type || '').toUpperCase()
    if (keyName === 'PRIMARY' || keyName.endsWith('_pkey') || rawType === 'PRIMARY') {
      return 'PRIMARY'
    }
    if (row.Non_unique === 0) {
      return 'UNIQUE'
    }
    return 'INDEX'
  }

  const normalizeIndexRows = (rows: any[]) => {
    const grouped = new Map<string, any>()
    rows.forEach((row: any, idx: number) => {
      const keyName = String(row.Key_name || '')
      if (!keyName) return
      const current = grouped.get(keyName)
      const columnName = String(row.Column_name || '').trim()
      const columnSeq = Number(row.Seq_in_index ?? row.seqno ?? idx)
      if (!current) {
        grouped.set(keyName, {
          Key_name: keyName,
          Non_unique: row.Non_unique,
          Index_type: normalizeIndexType(row),
          columns: columnName ? [{ name: columnName, seq: columnSeq }] : []
        })
        return
      }
      if (columnName && !current.columns.some((item: any) => item.name === columnName)) {
        current.columns.push({ name: columnName, seq: columnSeq })
      }
      if (current.Non_unique !== 0 && row.Non_unique === 0) {
        current.Non_unique = 0
      }
    })

    return Array.from(grouped.values()).map((row: any) => {
      const orderedColumns = row.columns
        .sort((a: any, b: any) => a.seq - b.seq)
        .map((item: any) => item.name)

      return {
        Key_name: row.Key_name,
        Non_unique: row.Non_unique,
        Index_type: row.Index_type,
        Column_name: orderedColumns.join(', '),
        columns: orderedColumns
      }
    })
  }

  const openAddColumnModal = () => {
    isEditColumn.value = false
    columnForm.value = { oldName: '', name: '', type: 'VARCHAR(255)', nullable: true, defaultValue: '', comment: '', autoIncrement: false, afterColumn: '' }
    showColumnModal.value = true
  }

  const openEditColumnModal = (row: any) => {
    isEditColumn.value = true
    const extra = String(row.Extra || '').toLowerCase()
    const isNullable = row.Null === 'YES' || row.Null === true || row.Null === 1
    const hasDefault = row.Default !== null && row.Default !== undefined
    columnForm.value = {
      oldName: row.Field,
      name: row.Field,
      type: row.Type,
      nullable: isNullable,
      defaultValue: hasDefault ? String(row.Default) : '',
      comment: String(row.Comment || ''),
      autoIncrement: extra.includes('auto_increment'),
      afterColumn: ''
    }
    showColumnModal.value = true
  }

  const dropColumn = async (row: any) => {
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
    const server = selectedServer.value
    const isPg = server?.type === 'postgresql'

    const sql = isPg
      ? `ALTER TABLE "${props.selectedTable}" DROP COLUMN "${row.Field}"`
      : `ALTER TABLE \`${props.selectedTable}\` DROP COLUMN \`${row.Field}\``

    try {
      const res = await execDBManagerSqlAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        sql
      })
      if (res.code === 0) {
        message.success('删除字段成功')
        emit.refresh()
        fetchTableIndexes()
      } else {
        message.error(res.message || '删除字段失败')
      }
    } catch {
      message.error('执行删除请求失败')
    }
  }

  const buildColumnDef = (type: string, nullable: boolean, defaultValue: string, autoIncrement: boolean, comment: string, isPg: boolean): string => {
    let def = type
    if (isPg) {
      // PostgreSQL handles nullable and default separately via ALTER COLUMN SET
      return def
    }
    if (!nullable) {
      def += ' NOT NULL'
    } else {
      def += ' NULL'
    }
    if (autoIncrement && !isPg) {
      def += ' AUTO_INCREMENT'
    }
    if (defaultValue !== '' && defaultValue !== null && defaultValue !== undefined) {
      // Quote string defaults, leave numeric/expression defaults unquoted
      const numVal = Number(defaultValue)
      const isNumeric = !Number.isNaN(numVal) && String(numVal) === defaultValue.trim()
      if (isNumeric) {
        def += ` DEFAULT ${numVal}`
      } else if (defaultValue.toUpperCase() === 'CURRENT_TIMESTAMP' || defaultValue.toUpperCase() === 'NOW()') {
        def += ` DEFAULT ${defaultValue}`
      } else {
        const escaped = defaultValue.replace(/'/g, "''")
        def += ` DEFAULT '${escaped}'`
      }
    }
    if (comment) {
      const escaped = comment.replace(/'/g, "''")
      def += ` COMMENT '${escaped}'`
    }
    return def
  }

  const buildAfterClause = (afterColumn: string, isPg: boolean): string => {
    if (isPg) return ''
    if (!afterColumn) return ''
    if (afterColumn === '__FIRST__') return ' FIRST'
    return ` AFTER \`${afterColumn}\``
  }

  const submitColumn = async () => {
    if (!columnForm.value.name || !columnForm.value.type) {
      message.warning('字段名和类型不能为空')
      return
    }

    const server = selectedServer.value
    const isPg = server?.type === 'postgresql'
    const table = props.selectedTable
    let sql = ''

    if (isEditColumn.value) {
      if (isPg) {
        const queries: string[] = []
        if (columnForm.value.oldName !== columnForm.value.name) {
          queries.push(`ALTER TABLE "${table}" RENAME COLUMN "${columnForm.value.oldName}" TO "${columnForm.value.name}"`)
        }
        if (columnForm.value.type) {
          queries.push(`ALTER TABLE "${table}" ALTER COLUMN "${columnForm.value.name}" TYPE ${columnForm.value.type} USING "${columnForm.value.name}"::${columnForm.value.type}`)
        }
        // Nullable
        if (columnForm.value.nullable) {
          queries.push(`ALTER TABLE "${table}" ALTER COLUMN "${columnForm.value.name}" DROP NOT NULL`)
        } else {
          queries.push(`ALTER TABLE "${table}" ALTER COLUMN "${columnForm.value.name}" SET NOT NULL`)
        }
        // Default
        if (columnForm.value.defaultValue !== '' && columnForm.value.defaultValue !== null && columnForm.value.defaultValue !== undefined) {
          queries.push(`ALTER TABLE "${table}" ALTER COLUMN "${columnForm.value.name}" SET DEFAULT '${columnForm.value.defaultValue.replace(/'/g, "''")}'`)
        } else if (columnForm.value.defaultValue === '') {
          // Only drop default if the original had one and user cleared it
          // We'll always set a default; empty means drop default via a separate query
        }
        // Comment
        if (columnForm.value.comment) {
          queries.push(`COMMENT ON COLUMN "${table}"."${columnForm.value.name}" IS '${columnForm.value.comment.replace(/'/g, "''")}'`)
        }
        sql = queries.join('; ')
      } else {
        const colDef = buildColumnDef(columnForm.value.type, columnForm.value.nullable, columnForm.value.defaultValue, columnForm.value.autoIncrement, columnForm.value.comment, false)
        const after = buildAfterClause(columnForm.value.afterColumn, false)
        sql = `ALTER TABLE \`${table}\` CHANGE COLUMN \`${columnForm.value.oldName}\` \`${columnForm.value.name}\` ${colDef}${after}`
      }
    } else {
      if (isPg) {
        let def = columnForm.value.type
        if (!columnForm.value.nullable) {
          def += ' NOT NULL'
        }
        if (columnForm.value.defaultValue !== '' && columnForm.value.defaultValue !== null && columnForm.value.defaultValue !== undefined) {
          def += ` DEFAULT '${columnForm.value.defaultValue.replace(/'/g, "''")}'`
        }
        sql = `ALTER TABLE "${table}" ADD COLUMN "${columnForm.value.name}" ${def}`
        if (columnForm.value.comment) {
          sql += `; COMMENT ON COLUMN "${table}"."${columnForm.value.name}" IS '${columnForm.value.comment.replace(/'/g, "''")}'`
        }
      } else {
        const colDef = buildColumnDef(columnForm.value.type, columnForm.value.nullable, columnForm.value.defaultValue, columnForm.value.autoIncrement, columnForm.value.comment, false)
        const after = buildAfterClause(columnForm.value.afterColumn, false)
        sql = `ALTER TABLE \`${table}\` ADD COLUMN \`${columnForm.value.name}\` ${colDef}${after}`
      }
    }

    submittingColumn.value = true
    try {
      const res = await execDBManagerSqlAPI({
        serverId: props.selectedServerId!,
        databaseName: props.selectedDatabase!,
        sql
      })
      if (res.code === 0) {
        message.success('操作成功')
        showColumnModal.value = false
        emit.refresh()
        fetchTableIndexes()
      } else {
        message.error(res.message || '操作失败')
      }
    } catch {
      message.error('执行失败')
    } finally {
      submittingColumn.value = false
    }
  }

  const fetchTableIndexes = async () => {
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
    const server = selectedServer.value
    if (!server) return

    loadingIndex.value = true
    let sql = ''
    if (server.type === 'mysql' || server.type === 'mariadb') {
      sql = `SHOW INDEX FROM \`${props.selectedTable}\``
    } else if (server.type === 'sqlite') {
      sql = `
        SELECT
          il.name AS "Key_name",
          CASE WHEN il.[unique] = 1 THEN 0 ELSE 1 END AS "Non_unique",
          ii.name AS "Column_name",
          il.origin AS "Index_type"
        FROM pragma_index_list('${props.selectedTable}') il
        LEFT JOIN pragma_index_info(il.name) ii
        ORDER BY il.seq, ii.seqno
      `
    } else {
      sql = `
        SELECT
          i.relname as "Key_name",
          ix.indisunique as "Non_unique",
          a.attname as "Column_name",
          am.amname as "Index_type"
        FROM
          pg_class t,
          pg_class i,
          pg_index ix,
          pg_attribute a,
          pg_am am
        WHERE
          t.oid = ix.indrelid
          and i.oid = ix.indexrelid
          and a.attrelid = t.oid
          and a.attnum = ANY(ix.indkey)
          and i.relam = am.oid
          and t.relkind = 'r'
          and t.relname = '${props.selectedTable}'
      `
    }

    try {
      const res = await execDBManagerSqlAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        sql
      })

      if (res.code === 0 && res.data && res.data.type === 'query') {
        if (server.type === 'postgresql') {
          indexData.value = normalizeIndexRows(res.data.rows.map((row: any) => ({
            Key_name: row.Key_name,
            Non_unique: row.Non_unique ? 0 : 1,
            Column_name: row.Column_name,
            Index_type: row.Key_name?.endsWith('_pkey') ? 'PRIMARY' : row.Index_type
          })) || [])
        } else {
          indexData.value = normalizeIndexRows(res.data.rows || [])
        }
      } else {
        indexData.value = []
      }
    } catch {
      indexData.value = []
      message.error('获取索引数据失败')
    } finally {
      loadingIndex.value = false
    }
  }

  const openAddIndexModal = () => {
    isEditIndex.value = false
    indexForm.value = { oldName: '', oldType: 'INDEX', oldColumns: [], name: '', type: 'INDEX', columns: [] }
    showIndexModal.value = true
  }

  const openEditIndexModal = (row: any) => {
    isEditIndex.value = true
    const normalizedType = normalizeIndexType(row)
    const columns = Array.isArray(row.columns)
      ? row.columns
      : String(row.Column_name || '')
          .split(',')
          .map((item: string) => item.trim())
          .filter(Boolean)

    indexForm.value = {
      oldName: row.Key_name,
      oldType: normalizedType,
      oldColumns: [...columns],
      name: normalizedType === 'PRIMARY' ? row.Key_name : row.Key_name,
      type: normalizedType,
      columns: [...columns]
    }
    showIndexModal.value = true
  }

  const dropIndex = async (row: any) => {
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
    const server = selectedServer.value
    const isPg = server?.type === 'postgresql'
    const isSqlite = server?.type === 'sqlite'
    const table = props.selectedTable
    const indexName = row.Key_name

    let sql = ''
    if (isPg) {
      sql = indexName.endsWith('_pkey')
        ? `ALTER TABLE "${table}" DROP CONSTRAINT "${indexName}"`
        : `DROP INDEX "${indexName}"`
    } else if (isSqlite) {
      sql = `DROP INDEX "${indexName}"`
    } else {
      sql = indexName === 'PRIMARY'
        ? `ALTER TABLE \`${table}\` DROP PRIMARY KEY`
        : `ALTER TABLE \`${table}\` DROP INDEX \`${indexName}\``
    }

    try {
      const res = await execDBManagerSqlAPI({
        serverId: props.selectedServerId,
        databaseName: props.selectedDatabase,
        sql
      })
      if (res.code === 0) {
        message.success('删除索引成功')
        fetchTableIndexes()
      } else {
        message.error(res.message || '删除索引失败')
      }
    } catch {
      message.error('执行删除请求失败')
    }
  }

  const submitIndex = async () => {
    if (!indexForm.value.type || indexForm.value.columns.length === 0) {
      message.warning('请选择索引类型和相关字段')
      return
    }

    const server = selectedServer.value
    const isPg = server?.type === 'postgresql'
    const isSqlite = server?.type === 'sqlite'
    const table = props.selectedTable
    const colStr = indexForm.value.columns.map(c => isPg ? `"${c}"` : `\`${c}\``).join(', ')
    const indexName = indexForm.value.name || `${table}_${indexForm.value.columns.join('_')}_idx`
    const statements: string[] = []

    if (isEditIndex.value) {
      const oldName = indexForm.value.oldName
      const oldType = indexForm.value.oldType
      if (oldType === 'PRIMARY' && isSqlite) {
        message.warning('SQLite 暂不支持修改主键索引')
        return
      }
      if (isPg) {
        statements.push(
          oldType === 'PRIMARY' || oldName.endsWith('_pkey')
            ? `ALTER TABLE "${table}" DROP CONSTRAINT "${oldName}"`
            : `DROP INDEX "${oldName}"`
        )
      } else if (isSqlite) {
        statements.push(`DROP INDEX "${oldName}"`)
      } else {
        statements.push(
          oldType === 'PRIMARY' || oldName === 'PRIMARY'
            ? `ALTER TABLE \`${table}\` DROP PRIMARY KEY`
            : `ALTER TABLE \`${table}\` DROP INDEX \`${oldName}\``
        )
      }
    }

    if (isPg) {
      if (indexForm.value.type === 'PRIMARY') {
        statements.push(`ALTER TABLE "${table}" ADD PRIMARY KEY (${colStr})`)
      } else if (indexForm.value.type === 'UNIQUE') {
        statements.push(`CREATE UNIQUE INDEX "${indexName}" ON "${table}" (${colStr})`)
      } else {
        statements.push(`CREATE INDEX "${indexName}" ON "${table}" (${colStr})`)
      }
    } else if (isSqlite) {
      if (indexForm.value.type === 'PRIMARY') {
        message.warning('SQLite 暂不支持直接新增主键索引')
        return
      }
      if (indexForm.value.type === 'UNIQUE') {
        statements.push(`CREATE UNIQUE INDEX "${indexName}" ON "${table}" (${indexForm.value.columns.map(c => `"${c}"`).join(', ')})`)
      } else {
        statements.push(`CREATE INDEX "${indexName}" ON "${table}" (${indexForm.value.columns.map(c => `"${c}"`).join(', ')})`)
      }
    } else {
      if (indexForm.value.type === 'PRIMARY') {
        statements.push(`ALTER TABLE \`${table}\` ADD PRIMARY KEY (${colStr})`)
      } else if (indexForm.value.type === 'UNIQUE') {
        statements.push(`ALTER TABLE \`${table}\` ADD UNIQUE INDEX \`${indexName}\` (${colStr})`)
      } else {
        statements.push(`ALTER TABLE \`${table}\` ADD INDEX \`${indexName}\` (${colStr})`)
      }
    }

    const sql = statements.join('; ')

    submittingIndex.value = true
    try {
      const res = await execDBManagerSqlAPI({
        serverId: props.selectedServerId!,
        databaseName: props.selectedDatabase!,
        sql
      })
      if (res.code === 0) {
        message.success(isEditIndex.value ? '修改索引成功' : '创建索引成功')
        showIndexModal.value = false
        fetchTableIndexes()
      } else {
        message.error(res.message || (isEditIndex.value ? '修改索引失败' : '创建索引失败'))
      }
    } catch {
      message.error('执行失败')
    } finally {
      submittingIndex.value = false
    }
  }

  const indexColumns = [
    {
      title: '操作',
      key: 'actions',
      width: 140,
      render(row: any) {
        return h('div', { class: 'flex gap-2' }, [
          h(NButton, { size: 'tiny', type: 'primary', ghost: true, onClick: () => openEditIndexModal(row) }, { default: () => '修改' }),
          h(NPopconfirm, { onPositiveClick: () => dropIndex(row) }, {
            trigger: () => h(NButton, { size: 'tiny', type: 'error', ghost: true }, { default: () => '删除' }),
            default: () => `确定要删除索引 ${row.Key_name} 吗？`
          })
        ])
      }
    },
    { title: '键名', key: 'Key_name', width: 150 },
    { title: '类型', key: 'Index_type', width: 100 },
    {
      title: '唯一',
      key: 'Non_unique',
      width: 100,
      render: (row: any) => row.Non_unique === 0 ? '是' : '否'
    },
    { title: '字段', key: 'Column_name', width: 150 }
  ]

  const structureColumns = computed(() => {
    if (!props.structureData || props.structureData.length === 0) return []
    const firstRow = props.structureData[0]
    const keys = Object.keys(firstRow)

    const actionCol = {
      title: '操作',
      key: 'actions',
      fixed: 'left' as const,
      width: 120,
      render(row: any) {
        return h('div', { class: 'flex gap-2' }, [
          h(NButton, { size: 'tiny', type: 'primary', ghost: true, onClick: () => openEditColumnModal(row) }, { default: () => '修改' }),
          h(NPopconfirm, { onPositiveClick: () => dropColumn(row) }, {
            trigger: () => h(NButton, { size: 'tiny', type: 'error', ghost: true }, { default: () => '删除' }),
            default: () => `确定要删除字段 ${row.Field} 吗？`
          })
        ])
      }
    }

    return [
      actionCol,
      ...keys.map((col, index) => ({
        title: col,
        key: col,
        ellipsis: { tooltip: true as const },
        className: index === 0 ? 'db-structure-primary-col' : undefined
      }))
    ]
  })

  // ═══════════════════════════ 外键管理 ═══════════════════════════

  const fkData = ref<any[]>([])
  const loadingFk = ref(false)

  const fkSummary = computed(() => ({
    total: fkData.value.length,
    cascadeDelete: fkData.value.filter((r: any) => r.DELETE_RULE === 'CASCADE').length
  }))

  const showFkModal = ref(false)
  const submittingFk = ref(false)
  const fkForm = ref({
    name: '',
    column: '',
    refTable: '',
    refColumn: '',
    onDelete: 'NO ACTION',
    onUpdate: 'NO ACTION'
  })

  const refTables = ref<{ label: string; value: string }[]>([])
  const refColumns = ref<{ label: string; value: string }[]>([])
  const loadingRefTables = ref(false)
  const loadingRefColumns = ref(false)

  const fkRuleOptions = [
    { label: 'NO ACTION', value: 'NO ACTION' },
    { label: 'RESTRICT', value: 'RESTRICT' },
    { label: 'CASCADE', value: 'CASCADE' },
    { label: 'SET NULL', value: 'SET NULL' },
    { label: 'SET DEFAULT', value: 'SET DEFAULT' }
  ]

  const fetchTableForeignKeys = async () => {
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
    const server = selectedServer.value
    if (!server) return

    loadingFk.value = true
    let sql = ''
    const t = props.selectedTable
    const db = props.selectedDatabase

    if (server.type === 'mysql' || server.type === 'mariadb') {
      sql = `SELECT kcu.CONSTRAINT_NAME, kcu.COLUMN_NAME, kcu.REFERENCED_TABLE_NAME, kcu.REFERENCED_COLUMN_NAME, rc.UPDATE_RULE, rc.DELETE_RULE FROM information_schema.KEY_COLUMN_USAGE kcu JOIN information_schema.REFERENTIAL_CONSTRAINTS rc ON kcu.CONSTRAINT_NAME = rc.CONSTRAINT_NAME AND kcu.CONSTRAINT_SCHEMA = rc.CONSTRAINT_SCHEMA WHERE kcu.TABLE_SCHEMA = '${db.replace(/'/g, "''")}' AND kcu.TABLE_NAME = '${t.replace(/'/g, "''")}' AND kcu.REFERENCED_TABLE_NAME IS NOT NULL`
    } else if (server.type === 'postgresql') {
      sql = `SELECT tc.constraint_name, kcu.column_name, ccu.table_name AS referenced_table_name, ccu.column_name AS referenced_column_name, rc.update_rule, rc.delete_rule FROM information_schema.table_constraints tc JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name AND tc.table_schema = kcu.table_schema JOIN information_schema.constraint_column_usage ccu ON tc.constraint_name = ccu.constraint_name AND tc.table_schema = ccu.table_schema JOIN information_schema.referential_constraints rc ON tc.constraint_name = rc.constraint_name AND tc.table_schema = rc.constraint_schema WHERE tc.constraint_type = 'FOREIGN KEY' AND tc.table_schema = 'public' AND tc.table_name = '${t.replace(/'/g, "''")}'`
    } else if (server.type === 'sqlite') {
      sql = `PRAGMA foreign_key_list('${t.replace(/'/g, "''")}')`
    }

    if (!sql) { loadingFk.value = false; return }

    try {
      const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId, databaseName: props.selectedDatabase, sql })
      if (res.code === 0 && res.data && res.data.type === 'query') {
        if (server.type === 'sqlite') {
          fkData.value = (res.data.rows || []).map((row: any) => ({
            CONSTRAINT_NAME: row.id ? `fk_${row.seq}` : '-',
            COLUMN_NAME: row.from || row.col || row.column || `col_${row.seq}`,
            REFERENCED_TABLE_NAME: row.table,
            REFERENCED_COLUMN_NAME: row.to || row.foreign_col || row.ref_col || '',
            UPDATE_RULE: row.on_update || 'NO ACTION',
            DELETE_RULE: row.on_delete || 'NO ACTION'
          }))
        } else {
          fkData.value = res.data.rows || []
        }
      } else {
        fkData.value = []
      }
    } catch {
      fkData.value = []
    } finally {
      loadingFk.value = false
    }
  }

  const openAddFkModal = async () => {
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
    showFkModal.value = true
    fkForm.value = { name: '', column: '', refTable: '', refColumn: '', onDelete: 'NO ACTION', onUpdate: 'NO ACTION' }
    refTables.value = []
    refColumns.value = []

    const server = selectedServer.value
    if (!server) return

    loadingRefTables.value = true
    let sql = ''
    if (server.type === 'mysql' || server.type === 'mariadb') {
      sql = 'SHOW TABLES'
    } else if (server.type === 'postgresql') {
      sql = "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'"
    } else if (server.type === 'sqlite') {
      sql = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
    }
    try {
      const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId, databaseName: props.selectedDatabase, sql })
      if (res.code === 0 && res.data && res.data.type === 'query') {
        const rows = res.data.rows || []
        const key = Object.keys(rows[0] || {})[0]
        refTables.value = rows.map((r: any) => ({ label: r[key] || Object.values(r)[0], value: r[key] || Object.values(r)[0] }))
      }
    } catch {
      // silently fail
    } finally {
      loadingRefTables.value = false
    }
  }

  const onRefTableChange = async (tableName: string) => {
    if (!tableName || !props.selectedServerId || !props.selectedDatabase) return
    const server = selectedServer.value
    if (!server) return

    fkForm.value.refColumn = ''
    loadingRefColumns.value = true
    let sql = ''
    if (server.type === 'mysql' || server.type === 'mariadb') {
      sql = `SHOW COLUMNS FROM \`${tableName.replace(/`/g, '')}\``
    } else if (server.type === 'postgresql') {
      sql = `SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = '${tableName.replace(/'/g, "''")}'`
    } else if (server.type === 'sqlite') {
      sql = `PRAGMA table_info('${tableName.replace(/'/g, "''")}')`
    }
    try {
      const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId, databaseName: props.selectedDatabase, sql })
      if (res.code === 0 && res.data && res.data.type === 'query') {
        const rows = res.data.rows || []
        if (server.type === 'mysql') {
          refColumns.value = rows.map((r: any) => ({ label: r.Field, value: r.Field }))
        } else if (server.type === 'postgresql') {
          refColumns.value = rows.map((r: any) => ({ label: r.column_name, value: r.column_name }))
        } else {
          const key = Object.keys(rows[0] || {})[0]
          refColumns.value = rows.map((r: any) => ({ label: r[key] || r.name, value: r[key] || r.name }))
        }
      }
    } catch {
      refColumns.value = []
    } finally {
      loadingRefColumns.value = false
    }
  }

  const dropForeignKey = async (row: any) => {
    if (!props.selectedServerId || !props.selectedDatabase || !props.selectedTable) return
    const server = selectedServer.value
    if (!server) return
    const table = props.selectedTable
    const cname = row.CONSTRAINT_NAME

    let sql = ''
    if (server.type === 'mysql' || server.type === 'mariadb') {
      sql = `ALTER TABLE \`${table}\` DROP FOREIGN KEY \`${cname}\``
    } else if (server.type === 'postgresql') {
      sql = `ALTER TABLE "${table}" DROP CONSTRAINT "${cname}"`
    } else {
      message.warning('SQLite 不支持直接删除外键约束')
      return
    }

    try {
      const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId, databaseName: props.selectedDatabase, sql })
      if (res.code === 0) {
        message.success('删除外键成功')
        fetchTableForeignKeys()
      } else {
        message.error(res.message || '删除外键失败')
      }
    } catch {
      message.error('执行删除请求失败')
    }
  }

  const submitForeignKey = async () => {
    if (!fkForm.value.column || !fkForm.value.refTable || !fkForm.value.refColumn) {
      message.warning('请选择字段、引用表和引用列')
      return
    }
    const server = selectedServer.value
    if (!server) return
    const isPg = server.type === 'postgresql'
    const q = isPg ? '"' : '`'
    const table = props.selectedTable
    const col = fkForm.value.column
    const rt = fkForm.value.refTable
    const rc = fkForm.value.refColumn
    const cname = fkForm.value.name || `fk_${table}_${col}`
    const onDel = fkForm.value.onDelete
    const onUpd = fkForm.value.onUpdate

    if (server.type === 'sqlite') {
      message.warning('SQLite 无法通过 ALTER TABLE 新增外键约束')
      return
    }

    const sql = `ALTER TABLE ${q}${table}${q} ADD CONSTRAINT ${q}${cname}${q} FOREIGN KEY (${q}${col}${q}) REFERENCES ${q}${rt}${q}(${q}${rc}${q}) ON DELETE ${onDel} ON UPDATE ${onUpd}`

    submittingFk.value = true
    try {
      const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId, databaseName: props.selectedDatabase, sql })
      if (res.code === 0) {
        message.success('添加外键成功')
        showFkModal.value = false
        fetchTableForeignKeys()
      } else {
        message.error(res.message || '添加外键失败')
      }
    } catch {
      message.error('执行失败')
    } finally {
      submittingFk.value = false
    }
  }

  const fkColumns = [
    {
      title: '操作',
      key: 'actions',
      width: 100,
      render(row: any) {
        return h(NPopconfirm, { onPositiveClick: () => dropForeignKey(row) }, {
          trigger: () => h(NButton, { size: 'tiny', type: 'error', ghost: true }, { default: () => '删除' }),
          default: () => `确定要删除外键 ${row.CONSTRAINT_NAME} 吗？`
        })
      }
    },
    { title: '约束名', key: 'CONSTRAINT_NAME', width: 160, ellipsis: { tooltip: true } as const },
    { title: '字段', key: 'COLUMN_NAME', width: 120 },
    { title: '引用表', key: 'REFERENCED_TABLE_NAME', width: 140 },
    { title: '引用列', key: 'REFERENCED_COLUMN_NAME', width: 120 },
    { title: '更新规则', key: 'UPDATE_RULE', width: 110 },
    { title: '删除规则', key: 'DELETE_RULE', width: 110 }
  ]

  watch(() => props.selectedTable, () => {
    if (props.selectedTable) {
      fetchTableIndexes()
      fetchTableForeignKeys()
    } else {
      indexData.value = []
      fkData.value = []
    }
  }, { immediate: true })

  onMounted(() => {
    if (props.selectedTable) {
      fetchTableIndexes()
    }
  })

  return {
    afterColumnOptions,
    columnForm,
    dropColumn,
    dropForeignKey,
    fetchTableForeignKeys,
    fetchTableIndexes,
    fieldSummary,
    fkColumns,
    fkData,
    fkForm,
    fkRuleOptions,
    fkSummary,
    indexColumns,
    indexColumnsOptions,
    indexData,
    indexForm,
    indexSummary,
    indexTypeOptions,
    isEditColumn,
    isEditIndex,
    loadingFk,
    loadingIndex,
    loadingRefColumns,
    loadingRefTables,
    onRefTableChange,
    openAddColumnModal,
    openAddFkModal,
    openAddIndexModal,
    openEditColumnModal,
    openEditIndexModal,
    refColumns,
    refTables,
    selectedServerLabel,
    showColumnModal,
    showFkModal,
    showIndexModal,
    structureColumns,
    submitColumn,
    submitForeignKey,
    submitIndex,
    submittingColumn,
    submittingFk,
    submittingIndex
  }
}
