console.log('app.js loaded');
console.log('Vue:', typeof Vue);
console.log('window.CRUD_CONFIG:', window.CRUD_CONFIG);

const { createApp } = Vue;

// 全局配置管理器
const ConfigManager = {
    config: null,
    
    loadConfig() {
        // 直接从页面注入的配置获取
        if (window.CRUD_CONFIG) {
            this.config = window.CRUD_CONFIG;
            return Promise.resolve(this.config);
        }
        
        // 如果没有注入配置，使用默认配置
        console.warn('No config injected, using defaults');
        this.config = {
            api_base_path: '/api',
            ui_base_path: '/crud-ui'
        };
        return Promise.resolve(this.config);
    },
    
    getApiUrl(path) {
        const basePath = this.config ? this.config.api_base_path : '/api';
        return basePath + (path.startsWith('/') ? path : '/' + path);
    }
};

// 等待DOM加载完成后再创建Vue应用
document.addEventListener('DOMContentLoaded', function() {
    console.log('DOM loaded, about to create Vue app');
    const appElement = document.querySelector('#app');
    console.log('Target element:', appElement);
    console.log('Target element innerHTML length:', appElement ? appElement.innerHTML.length : 'element not found');

    if (!appElement) {
        console.error('App element not found!');
        return;
    }

    // 保存原始HTML内容作为模板
    const template = appElement.innerHTML;
    console.log('Saved template, length:', template.length);

    const app = createApp({
        template: template.trim() ? template : '<div>No template found</div>',
    data() {
        return {
            // 应用数据
            connections: {},
            configs: [],
            selectedConfig: null,
            selectedConnectionId: '',
            saving: false,
            creating: false,
            message: '',
            messageType: 'success',
            sqlFields: [], // 解析到的SQL字段
            newSqlFields: [], // 新建配置的SQL字段
            validationErrors: [], // 验证错误
            newConfig: {
                name: '',
                table_name: '',
                connection_id: '',
                create_statement: '',
                query_pagination: true,
                displayFields: [],
                searchFields: [],
                creatableFields: [],
                updatableFields: []
            },
            dragState: {
                draggedIndex: null,
                draggedFieldType: null
            }
        }
    },
    computed: {
        filteredConfigs() {
            if (!this.selectedConnectionId) return this.configs;
            return this.configs.filter(config => config.connection_id === this.selectedConnectionId);
        }
    },
    mounted() {
        console.log('Vue mounted() called');
        try {
            // 加载配置并初始化应用
            console.log('Calling initializeApp()');
            this.initializeApp();
            console.log('initializeApp() completed');
            
            // 初始化Bootstrap tooltips
            this.$nextTick(() => {
                console.log('nextTick callback called');
                const tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
                tooltipTriggerList.map(function (tooltipTriggerEl) {
                    return new bootstrap.Tooltip(tooltipTriggerEl);
                });
                console.log('Bootstrap tooltips initialized');
            });
        } catch (error) {
            console.error('Error in mounted():', error);
        }
    },
    methods: {
        // 初始化应用
        initializeApp() {
            try {
                ConfigManager.loadConfig();
                this.loadConnections();
                this.loadConfigs();
            } catch (error) {
                console.error('Failed to initialize app:', error);
                // 即使配置加载失败，也尝试加载数据
                this.loadConnections();
                this.loadConfigs();
            }
        },
        
        // 提取表名的方法
        extractTableName(sql) {
            if (!sql) return '';
            // 匹配 CREATE TABLE 语句中的表名
            const patterns = [
                /create\s+table\s+(?:if\s+not\s+exists\s+)?(?:public\.)?([a-zA-Z_][a-zA-Z0-9_]*)/i,
                /create\s+table\s+(?:if\s+not\s+exists\s+)?`?([a-zA-Z_][a-zA-Z0-9_]*)`?/i
            ];
            
            for (const pattern of patterns) {
                const match = sql.match(pattern);
                if (match && match[1]) {
                    return match[1].toLowerCase();
                }
            }
            return '';
        },
        
        // 当建表语句变化时自动提取表名和字段
        onCreateStatementChange() {
            if (this.selectedConfig && this.selectedConfig.create_statement) {
                const tableName = this.extractTableName(this.selectedConfig.create_statement);
                if (tableName && (!this.selectedConfig.name || !this.selectedConfig.table_name)) {
                    if (!this.selectedConfig.name) {
                        this.selectedConfig.name = tableName;
                    }
                    if (!this.selectedConfig.table_name) {
                        this.selectedConfig.table_name = tableName;
                    }
                }
                
                // 解析SQL字段
                this.sqlFields = this.parseSQLFields(this.selectedConfig.create_statement);
                
                // 自动初始化所有字段配置（如果当前配置为空）
                this.initializeFieldConfigurations();
                
                this.validateFields();
            }
        },
        
        // 新建配置时的建表语句变化处理
        onNewConfigCreateStatementChange() {
            if (this.newConfig.create_statement) {
                const tableName = this.extractTableName(this.newConfig.create_statement);
                if (tableName) {
                    if (!this.newConfig.name) {
                        this.newConfig.name = tableName;
                    }
                    if (!this.newConfig.table_name) {
                        this.newConfig.table_name = tableName;
                    }
                }
                
                // 解析SQL字段
                this.newSqlFields = this.parseSQLFields(this.newConfig.create_statement);
                
                // 自动初始化新配置的所有字段配置
                this.initializeNewConfigFields();
            }
        },
        
        // 解析SQL建表语句，提取字段信息
        parseSQLFields(sql) {
            if (!sql) return [];
            
            const fields = [];
            try {
                // 匹配CREATE TABLE语句中的字段定义
                const createTableMatch = sql.match(/create\s+table\s+(?:if\s+not\s+exists\s+)?(?:\w+\.)?(\w+)\s*\(([^;]+)/i);
                if (!createTableMatch) return fields;
                
                const fieldsText = createTableMatch[2];
                
                // 按逗号分割，但要考虑括号内的逗号
                const fieldDefs = this.splitByCommaOutsideParentheses(fieldsText);
                
                fieldDefs.forEach(fieldDef => {
                    const trimmed = fieldDef.trim();
                    // 跳过纯约束定义，但不跳过包含字段定义的行
                    if (!trimmed || 
                        trimmed.toLowerCase().startsWith('constraint') || 
                        trimmed.toLowerCase().startsWith('primary key') ||
                        trimmed.toLowerCase().startsWith('foreign key') ||
                        trimmed.toLowerCase().startsWith('unique') ||
                        trimmed.toLowerCase().startsWith('check') ||
                        trimmed.toLowerCase().startsWith('index')) {
                        return;
                    }
                    
                    // 匹配字段名和类型，改进正则表达式以处理更复杂的定义
                    const fieldMatch = trimmed.match(/^(\w+)\s+(\w+(?:\s*\([^)]*\))?)/i);
                    if (fieldMatch) {
                        fields.push({
                            name: fieldMatch[1].toLowerCase(),
                            type: fieldMatch[2].toLowerCase().replace(/\s+/g, '')
                        });
                    }
                });
            } catch (error) {
                console.warn('SQL解析失败:', error);
            }
            
            return fields;
        },
        
        // 分割字段定义，忽略括号内的逗号
        splitByCommaOutsideParentheses(text) {
            const result = [];
            let current = '';
            let parenthesesCount = 0;
            
            for (let i = 0; i < text.length; i++) {
                const char = text[i];
                if (char === '(') {
                    parenthesesCount++;
                } else if (char === ')') {
                    parenthesesCount--;
                } else if (char === ',' && parenthesesCount === 0) {
                    result.push(current.trim());
                    current = '';
                    continue;
                }
                current += char;
            }
            
            if (current.trim()) {
                result.push(current.trim());
            }
            
            return result;
        },
        
        // 验证字段是否有效
        isValidField(fieldName) {
            if (!fieldName) return true; // 空字段不算错误
            if (!Array.isArray(this.sqlFields)) return true; // 如果sqlFields不是数组，不进行验证
            return this.sqlFields.some(field => field && field.name === fieldName.toLowerCase());
        },
        
        // 验证所有字段
        validateFields() {
            this.validationErrors = [];
            
            if (!this.selectedConfig) return;
            
            // 确保sqlFields是数组
            if (!Array.isArray(this.sqlFields)) {
                this.sqlFields = [];
                return;
            }
            
            // 检查展示字段
            if (Array.isArray(this.selectedConfig.displayFields)) {
                this.selectedConfig.displayFields.forEach((field, index) => {
                    if (field && field.field && !this.isValidField(field.field)) {
                        this.validationErrors.push(`展示字段 ${field.field} 不存在`);
                    }
                });
            }
            
            // 检查搜索字段
            if (Array.isArray(this.selectedConfig.searchFields)) {
                this.selectedConfig.searchFields.forEach((field, index) => {
                    if (field && field.field && !this.isValidField(field.field)) {
                        this.validationErrors.push(`搜索字段 ${field.field} 不存在`);
                    }
                });
            }
            
            // 检查可创建字段
            if (Array.isArray(this.selectedConfig.creatableFields)) {
                this.selectedConfig.creatableFields.forEach((field, index) => {
                    if (field && field.field && !this.isValidField(field.field)) {
                        this.validationErrors.push(`创建字段 ${field.field} 不存在`);
                    }
                    // 验证不可编辑字段必须有默认值
                    if (field && field.type === 'readonly' && !field.default_type) {
                        this.validationErrors.push(`创建字段 ${field.field} 为不可编辑类型，必须配置默认值`);
                    }
                    // 验证固定值类型必须有默认值内容
                    if (field && field.type === 'readonly' && field.default_type === 'fixed' && !field.default_value) {
                        this.validationErrors.push(`创建字段 ${field.field} 的固定默认值不能为空`);
                    }
                });
            }
            
            // 检查可更新字段
            if (Array.isArray(this.selectedConfig.updatableFields)) {
                this.selectedConfig.updatableFields.forEach((field, index) => {
                    if (field && field.field && !this.isValidField(field.field)) {
                        this.validationErrors.push(`更新字段 ${field.field} 不存在`);
                    }
                    // 验证不可编辑字段必须有默认值
                    if (field && field.type === 'readonly' && !field.default_type) {
                        this.validationErrors.push(`更新字段 ${field.field} 为不可编辑类型，必须配置默认值`);
                    }
                    // 验证固定值类型必须有默认值内容
                    if (field && field.type === 'readonly' && field.default_type === 'fixed' && !field.default_value) {
                        this.validationErrors.push(`更新字段 ${field.field} 的固定默认值不能为空`);
                    }
                });
            }
        },

        // SQL格式化功能
        formatSQL(configType) {
            try {
                // 检查sql-formatter是否加载
                if (typeof sqlFormatter === 'undefined') {
                    this.message = 'SQL格式化库未加载，请刷新页面重试';
                    this.messageType = 'error';
                    setTimeout(() => { this.message = ''; }, 3000);
                    return;
                }

                let sqlStatement = '';
                if (configType === 'selectedConfig' && this.selectedConfig) {
                    sqlStatement = this.selectedConfig.create_statement;
                } else if (configType === 'newConfig') {
                    sqlStatement = this.newConfig.create_statement;
                }

                if (!sqlStatement || !sqlStatement.trim()) {
                    this.message = 'SQL语句为空，无法格式化';
                    this.messageType = 'error';
                    setTimeout(() => { this.message = ''; }, 3000);
                    return;
                }

                // 使用sql-formatter格式化SQL (v4.x API)
                const formatted = sqlFormatter.format(sqlStatement, {
                    indent: '  ',  // 使用2个空格缩进
                    uppercase: true  // 大写关键字
                });

                // 更新对应的配置
                if (configType === 'selectedConfig' && this.selectedConfig) {
                    this.selectedConfig.create_statement = formatted;
                    // 触发重新解析
                    this.onCreateStatementChange();
                } else if (configType === 'newConfig') {
                    this.newConfig.create_statement = formatted;
                    // 触发重新解析
                    this.onNewConfigCreateStatementChange();
                }

                this.message = 'SQL格式化成功';
                this.messageType = 'success';
                setTimeout(() => { this.message = ''; }, 2000);

            } catch (error) {
                console.error('SQL格式化失败:', error);
                this.message = 'SQL格式化失败: ' + error.message;
                this.messageType = 'error';
                setTimeout(() => { this.message = ''; }, 3000);
            }
        },

        // 自动格式化SQL（在配置加载时调用）
        autoFormatSQL(sqlStatement) {
            if (!sqlStatement || !sqlStatement.trim()) {
                return sqlStatement;
            }

            // 检查sql-formatter是否加载
            if (typeof sqlFormatter === 'undefined') {
                console.warn('SQL格式化库未加载，跳过自动格式化');
                return sqlStatement;
            }

            try {
                // 检查SQL是否已经格式化（简单检测：是否包含适当的换行和缩进）
                const lines = sqlStatement.split('\n');
                const hasProperFormatting = lines.length > 1 && 
                    lines.some(line => line.trim() !== line && line.startsWith('  '));
                
                // 如果已经有较好的格式化，就不自动格式化
                if (hasProperFormatting) {
                    return sqlStatement;
                }

                // 自动格式化 (v4.x API)
                return sqlFormatter.format(sqlStatement, {
                    indent: '  ',  // 使用2个空格缩进
                    uppercase: true  // 大写关键字
                });
            } catch (error) {
                console.warn('自动格式化SQL失败，保持原样:', error);
                return sqlStatement;
            }
        },
        
        // 转换JSON数据为表单数据
        convertJSONToFormData() {
            if (!this.selectedConfig) return;
            
            // 处理展示字段
            if (!Array.isArray(this.selectedConfig.displayFields)) {
                this.selectedConfig.displayFields = [];
            }
            if (this.selectedConfig.query_display_fields) {
                try {
                    const parsed = JSON.parse(this.selectedConfig.query_display_fields);
                    if (Array.isArray(parsed)) {
                        this.selectedConfig.displayFields = parsed;
                    }
                } catch (e) {
                    console.warn('展示字段JSON解析失败:', e);
                }
            }
            
            // 处理搜索字段 - 确保不覆盖已存在的数组
            if (!Array.isArray(this.selectedConfig.searchFields)) {
                this.selectedConfig.searchFields = [];
            }
            if (this.selectedConfig.query_search_fields) {
                try {
                    const parsed = JSON.parse(this.selectedConfig.query_search_fields);
                    if (Array.isArray(parsed)) {
                        this.selectedConfig.searchFields = parsed;
                    }
                } catch (e) {
                    console.warn('搜索字段JSON解析失败:', e);
                }
            }
            
            // 处理可创建字段
            if (!Array.isArray(this.selectedConfig.creatableFields)) {
                this.selectedConfig.creatableFields = [];
            }
            if (this.selectedConfig.create_creatable_fields) {
                try {
                    const parsed = JSON.parse(this.selectedConfig.create_creatable_fields);
                    if (Array.isArray(parsed)) {
                        this.selectedConfig.creatableFields = parsed;
                    }
                } catch (e) {
                    console.warn('可创建字段JSON解析失败:', e);
                }
            }
            
            // 处理可更新字段
            if (!Array.isArray(this.selectedConfig.updatableFields)) {
                this.selectedConfig.updatableFields = [];
            }
            if (this.selectedConfig.update_updatable_fields) {
                try {
                    const parsed = JSON.parse(this.selectedConfig.update_updatable_fields);
                    if (Array.isArray(parsed)) {
                        // 如果是新格式（对象数组），保持原样；如果是旧格式（字符串数组），转换为对象数组
                        if (parsed.length > 0 && typeof parsed[0] === 'string') {
                            this.selectedConfig.updatableFields = parsed.map(field => ({
                                field: field,
                                label: '',
                                type: 'text',
                                required: false
                            }));
                        } else {
                            this.selectedConfig.updatableFields = parsed;
                        }
                    }
                } catch (e) {
                    console.warn('可更新字段JSON解析失败:', e);
                }
            }
        },
        
        // 清理数据（将表单数据转换为JSON格式）
        cleanData(data) {
            const cleaned = {...data};
            
            // 转换展示字段
            if (cleaned.displayFields && Array.isArray(cleaned.displayFields)) {
                const validFields = cleaned.displayFields.filter(field => field.field && field.field.trim());
                cleaned.query_display_fields = validFields.length > 0 ? JSON.stringify(validFields) : '';
                
                // 从展示字段中提取可排序字段
                const sortableFields = validFields.filter(field => field.sortable).map(field => field.field);
                cleaned.query_sortable_fields = sortableFields.length > 0 ? JSON.stringify(sortableFields) : '';
            }
            
            // 转换搜索字段
            if (cleaned.searchFields && Array.isArray(cleaned.searchFields)) {
                const validFields = cleaned.searchFields.filter(field => field.field && field.field.trim());
                cleaned.query_search_fields = validFields.length > 0 ? JSON.stringify(validFields) : '';
            }
            
            // 转换可创建字段
            if (cleaned.creatableFields && Array.isArray(cleaned.creatableFields)) {
                const validFields = cleaned.creatableFields.filter(field => field.field && field.field.trim());
                cleaned.create_creatable_fields = validFields.length > 0 ? JSON.stringify(validFields) : '';
            }
            
            // 转换可更新字段
            if (cleaned.updatableFields && Array.isArray(cleaned.updatableFields)) {
                const validFields = cleaned.updatableFields.filter(field => field.field && field.field.trim());
                cleaned.update_updatable_fields = validFields.length > 0 ? JSON.stringify(validFields) : '';
            }
            
            // 清理SQL字段
            if (cleaned.create_statement) {
                cleaned.create_statement = cleaned.create_statement.replace(/\s+/g, ' ').trim();
            }
            
            // 删除表单专用字段
            delete cleaned.displayFields;
            delete cleaned.searchFields;
            delete cleaned.creatableFields;
            delete cleaned.updatableFields;
            
            return cleaned;
        },
        
        async loadConnections() {
            try {
                const response = await axios.get(ConfigManager.getApiUrl('/connections'));
                this.connections = response.data.data;
            } catch (error) {
                console.error('Failed to load connections:', error);
            }
        },
        
        async loadConfigs() {
            try {
                const response = await axios.get(ConfigManager.getApiUrl('/configs'));
                this.configs = response.data.data;
            } catch (error) {
                console.error('Failed to load configs:', error);
            }
        },
        
        selectConfig(config) {
            this.selectedConfig = {...config};
            this.message = '';
            
            // 确保数组字段被正确初始化
            this.selectedConfig.displayFields = this.selectedConfig.displayFields || [];
            this.selectedConfig.searchFields = this.selectedConfig.searchFields || [];
            this.selectedConfig.creatableFields = this.selectedConfig.creatableFields || [];
            this.selectedConfig.updatableFields = this.selectedConfig.updatableFields || [];
            
            // 自动格式化SQL语句
            if (this.selectedConfig.create_statement) {
                this.selectedConfig.create_statement = this.autoFormatSQL(this.selectedConfig.create_statement);
            }
            
            // 转换JSON字段为表单数据
            this.convertJSONToFormData();
            
            // 解析SQL字段
            this.sqlFields = this.parseSQLFields(this.selectedConfig.create_statement);
            
            // 初始化字段配置，确保包含所有字段
            this.initializeFieldConfigurations();
            
            this.validateFields();
            
            // 重新初始化 tooltips
            this.reinitializeTooltips();
        },
        
        filterByConnection() {
            this.selectedConfig = null;
            this.message = '';
        },
        
        async saveConfig() {
            if (!this.selectedConfig) return;
            
            // 验证字段
            this.validateFields();
            if (this.validationErrors.length > 0) {
                this.message = '配置中有无效字段，请检查并修正后再保存：\n' + this.validationErrors.join('\n');
                this.messageType = 'error';
                return;
            }
            
            this.saving = true;
            try {
                const cleanedConfig = this.cleanData(this.selectedConfig);
                await axios.put(ConfigManager.getApiUrl('/configs/' + this.selectedConfig.id), cleanedConfig);
                this.message = '配置保存成功';
                this.messageType = 'success';
                await this.loadConfigs();
            } catch (error) {
                this.message = '保存失败: ' + (error.response?.data?.error || error.message);
                this.messageType = 'error';
            } finally {
                this.saving = false;
                setTimeout(() => { this.message = ''; }, 3000);
            }
        },
        
        async createConfig() {
            this.creating = true;
            try {
                const cleanedConfig = this.cleanData(this.newConfig);
                await axios.post(ConfigManager.getApiUrl('/configs'), cleanedConfig);
                bootstrap.Modal.getInstance(document.getElementById('configModal')).hide();
                this.resetNewConfig();
                await this.loadConfigs();
            } catch (error) {
                alert('创建失败: ' + (error.response?.data?.error || error.message));
            } finally {
                this.creating = false;
            }
        },
        
        async deleteConfig() {
            if (!this.selectedConfig || !confirm('确定要删除这个配置吗？')) return;
            
            try {
                await axios.delete(ConfigManager.getApiUrl('/configs/' + this.selectedConfig.id));
                this.selectedConfig = null;
                await this.loadConfigs();
            } catch (error) {
                alert('删除失败: ' + (error.response?.data?.error || error.message));
            }
        },
        
        async testConnection(connectionId) {
            try {
                await axios.post(ConfigManager.getApiUrl('/connections/' + connectionId + '/test'));
                this.message = '数据库连接测试成功';
                this.messageType = 'success';
            } catch (error) {
                this.message = '连接测试失败: ' + (error.response?.data?.error || error.message);
                this.messageType = 'error';
            } finally {
                setTimeout(() => { this.message = ''; }, 3000);
            }
        },

        async testConnectionWithTable(connectionId, tableName) {
            try {
                await axios.post(ConfigManager.getApiUrl('/connections/' + connectionId + '/test-table'), {
                    table_name: tableName
                });
                this.message = `表 '${tableName}' 访问测试成功`;
                this.messageType = 'success';
            } catch (error) {
                this.message = '表访问测试失败: ' + (error.response?.data?.error || error.message);
                this.messageType = 'error';
            } finally {
                setTimeout(() => { this.message = ''; }, 3000);
            }
        },

        async testConfigConnection(configId) {
            try {
                await axios.post(ConfigManager.getApiUrl('/configs/' + configId + '/test'));
                this.message = '配置连接和表测试成功';
                this.messageType = 'success';
            } catch (error) {
                this.message = '配置测试失败: ' + (error.response?.data?.error || error.message);
                this.messageType = 'error';
            } finally {
                setTimeout(() => { this.message = ''; }, 3000);
            }
        },
        
        showConfigModal() {
            this.resetNewConfig();
        },
        
        resetNewConfig() {
            this.newConfig = {
                name: '',
                table_name: '',
                connection_id: '',
                create_statement: '',
                query_pagination: true,
                displayFields: [],
                searchFields: [],
                creatableFields: [],
                updatableFields: []
            };
        },
        
        // 重新初始化 tooltips
        reinitializeTooltips() {
            this.$nextTick(() => {
                // 销毁现有的 tooltips
                const existingTooltips = document.querySelectorAll('[data-bs-toggle="tooltip"]');
                existingTooltips.forEach(el => {
                    const tooltip = bootstrap.Tooltip.getInstance(el);
                    if (tooltip) {
                        tooltip.dispose();
                    }
                });
                
                // 重新初始化所有 tooltips
                const tooltipTriggerList = [].slice.call(document.querySelectorAll('[data-bs-toggle="tooltip"]'));
                tooltipTriggerList.map(function (tooltipTriggerEl) {
                    return new bootstrap.Tooltip(tooltipTriggerEl);
                });
            });
        },
        
        // 初始化字段配置（用于现有配置）
        initializeFieldConfigurations() {
            if (!this.sqlFields || this.sqlFields.length === 0) return;
            
            // 只在配置为空时初始化
            if (!this.selectedConfig.displayFields || this.selectedConfig.displayFields.length === 0) {
                this.selectedConfig.displayFields = this.sqlFields.map(field => ({
                    field: field.name,
                    label: this.getFieldLabel(field.name),
                    width: null,
                    sortable: field.name !== 'id', // id字段默认不可排序
                    searchable: false
                }));
            }
            
            if (!this.selectedConfig.searchFields || this.selectedConfig.searchFields.length === 0) {
                this.selectedConfig.searchFields = this.sqlFields
                    .filter(field => this.isSearchableType(field.type))
                    .map(field => ({
                        field: field.name,
                        label: this.getFieldLabel(field.name),
                        type: this.getDefaultSearchType(field.type),
                        dict_source: '',
                        dict_source_type: '' // 新增字段
                    }));
            } else {
                // 为现有搜索字段添加缺失的属性
                this.selectedConfig.searchFields.forEach(field => {
                    if (field.label === undefined) {
                        field.label = this.getFieldLabel(field.field) || field.field;
                    }
                    if (field.dict_source_type === undefined) {
                        // 根据现有的dict_source推断dict_source_type
                        if (!field.dict_source) {
                            field.dict_source_type = '';
                        } else if (this.sqlFields.some(sqlField => sqlField.name === field.dict_source)) {
                            // 如果dict_source是已知字段名
                            field.dict_source_type = field.dict_source;
                        } else {
                            // 否则认为是自定义
                            field.dict_source_type = 'custom';
                        }
                    }
                });
            }
            
            if (!this.selectedConfig.creatableFields || this.selectedConfig.creatableFields.length === 0) {
                this.selectedConfig.creatableFields = this.sqlFields.map(field => ({
                    field: field.name,
                    label: this.getFieldLabel(field.name),
                    type: this.getDefaultInputType(field.type),
                    required: field.name !== 'id', // id字段默认不必填
                    user_readonly: false, // 默认都是可以编辑的
                    default_type: field.name === 'id' ? 'auto_increment' : '',
                    default_value: ''
                }));
            } else {
                // 如果已有配置，确保所有SQL字段都包含
                const existingFieldNames = this.selectedConfig.creatableFields.map(f => f.field);
                const missingFields = this.sqlFields.filter(field => !existingFieldNames.includes(field.name));
                
                // 添加缺失的字段
                missingFields.forEach(field => {
                    this.selectedConfig.creatableFields.push({
                        field: field.name,
                        label: this.getFieldLabel(field.name),
                        type: this.getDefaultInputType(field.type),
                        required: field.name !== 'id',
                        user_readonly: false, // 默认都是可以编辑的
                        default_type: field.name === 'id' ? 'auto_increment' : '',
                        default_value: ''
                    });
                });
                
                // 为现有字段添加缺失的属性
                this.selectedConfig.creatableFields.forEach(field => {
                    if (field.user_readonly === undefined) {
                        // 如果是旧的user_input_required字段，转换为user_readonly（逻辑相反）
                        if (field.user_input_required !== undefined) {
                            field.user_readonly = !field.user_input_required;
                            delete field.user_input_required;
                        } else {
                            field.user_readonly = false; // 默认可编辑
                        }
                    }
                });
            }
            
            if (!this.selectedConfig.updatableFields || this.selectedConfig.updatableFields.length === 0) {
                // 确保主键字段在第一位，其他字段按顺序排列
                const primaryKeyFields = this.sqlFields.filter(field => field.name === 'id');
                const nonPrimaryKeyFields = this.sqlFields.filter(field => field.name !== 'id');
                const sortedFields = [...primaryKeyFields, ...nonPrimaryKeyFields];
                
                this.selectedConfig.updatableFields = sortedFields.map(field => ({
                    field: field.name,
                    label: this.getFieldLabel(field.name),
                    type: this.getDefaultInputType(field.type),
                    required: false,
                    is_primary_key: field.name === 'id', // 标记主键字段
                    user_input_required: field.name !== 'id', // 主键字段默认不需要用户输入
                    default_type: '',
                    default_value: ''
                }));
            } else {
                // 如果已有配置，确保所有SQL字段都包含，主键字段在第一位
                const existingFieldNames = this.selectedConfig.updatableFields.map(f => f.field);
                const missingFields = this.sqlFields.filter(field => !existingFieldNames.includes(field.name));
                
                // 为现有字段添加缺失的属性
                this.selectedConfig.updatableFields.forEach(field => {
                    if (field.is_primary_key === undefined) {
                        field.is_primary_key = field.field === 'id';
                    }
                    if (field.user_input_required === undefined) {
                        field.user_input_required = field.field !== 'id';
                    }
                    if (field.default_type === undefined) {
                        field.default_type = '';
                    }
                    if (field.default_value === undefined) {
                        field.default_value = '';
                    }
                });
                
                // 添加缺失的字段
                missingFields.forEach(field => {
                    this.selectedConfig.updatableFields.push({
                        field: field.name,
                        label: this.getFieldLabel(field.name),
                        type: this.getDefaultInputType(field.type),
                        required: false,
                        is_primary_key: field.name === 'id',
                        user_input_required: field.name !== 'id',
                        default_type: '',
                        default_value: ''
                    });
                });
                
                // 重新排序，确保主键字段在第一位
                const primaryKeyFields = this.selectedConfig.updatableFields.filter(field => field.is_primary_key);
                const nonPrimaryKeyFields = this.selectedConfig.updatableFields.filter(field => !field.is_primary_key);
                this.selectedConfig.updatableFields = [...primaryKeyFields, ...nonPrimaryKeyFields];
            }
            
            // 重新初始化 tooltips
            this.reinitializeTooltips();
        },
        
        // 初始化新配置的字段配置
        initializeNewConfigFields() {
            if (!this.newSqlFields || this.newSqlFields.length === 0) return;
            
            this.newConfig.displayFields = this.newSqlFields.map(field => ({
                field: field.name,
                label: this.getFieldLabel(field.name),
                width: null,
                sortable: field.name !== 'id',
                searchable: false
            }));
            
            this.newConfig.searchFields = this.newSqlFields
                .filter(field => this.isSearchableType(field.type))
                .map(field => ({
                    field: field.name,
                    label: this.getFieldLabel(field.name),
                    type: this.getDefaultSearchType(field.type),
                    dict_source: '',
                    dict_source_type: '' // 新增字段
                }));
                
            this.newConfig.creatableFields = this.newSqlFields.map(field => ({
                field: field.name,
                label: this.getFieldLabel(field.name),
                type: this.getDefaultInputType(field.type),
                required: field.name !== 'id',
                user_readonly: false, // 默认都是可以编辑的
                default_type: field.name === 'id' ? 'auto_increment' : '',
                default_value: ''
            }));
            
            // 确保主键字段在第一位，其他字段按顺序排列
            const primaryKeyFields = this.newSqlFields.filter(field => field.name === 'id');
            const nonPrimaryKeyFields = this.newSqlFields.filter(field => field.name !== 'id');
            const sortedFields = [...primaryKeyFields, ...nonPrimaryKeyFields];
            
            this.newConfig.updatableFields = sortedFields.map(field => ({
                field: field.name,
                label: this.getFieldLabel(field.name),
                type: this.getDefaultInputType(field.type),
                required: false,
                is_primary_key: field.name === 'id',
                user_input_required: field.name !== 'id',
                default_type: '',
                default_value: ''
            }));
        },
        
        // 获取字段标签（中文化处理）
        getFieldLabel(fieldName) {
            const labelMap = {
                'id': 'ID',
                'name': '姓名',
                'age': '年龄',
                'email': '邮箱',
                'phone': '电话',
                'address': '地址',
                'created_at': '创建时间',
                'updated_at': '更新时间'
            };
            return labelMap[fieldName] || fieldName;
        },
        
        // 判断字段类型是否可搜索
        isSearchableType(sqlType) {
            const type = sqlType.toLowerCase();
            return type.includes('varchar') || type.includes('text') || 
                   type.includes('char') || type.includes('int') ||
                   type.includes('numeric') || type.includes('decimal');
        },
        
        // 检查字段是否支持自增
        isAutoIncrementSupported(fieldName) {
            if (!this.selectedConfig || !this.selectedConfig.create_statement) {
                return false;
            }
            
            const sql = this.selectedConfig.create_statement.toLowerCase();
            
            // 检查是否包含自增关键字
            const autoIncrementPatterns = [
                'serial',           // PostgreSQL
                'bigserial',        // PostgreSQL  
                'auto_increment',   // MySQL
                'autoincrement',    // SQLite
                'identity',         // SQL Server
                'generated.*always.*as.*identity'  // SQL Standard
            ];
            
            // 检查字段定义中是否包含自增关键字
            for (const pattern of autoIncrementPatterns) {
                const regex = new RegExp(`\\b${fieldName}\\b.*${pattern}`, 'i');
                if (regex.test(sql)) {
                    return true;
                }
            }
            
            return false;
        },
        
        // 获取默认输入类型
        getDefaultInputType(sqlType) {
            const type = sqlType.toLowerCase();
            if (type.includes('int') || type.includes('numeric') || type.includes('decimal')) {
                return 'number';
            } else if (type.includes('date')) {
                return 'date';
            } else if (type.includes('timestamp')) {
                return 'datetime';
            } else if (type.includes('text')) {
                return 'textarea';
            } else {
                return 'text';
            }
        },
        
        // 拖拽排序相关方法
        onDragStart(event, index, fieldType) {
            this.dragState.draggedIndex = index;
            this.dragState.draggedFieldType = fieldType;
            event.target.classList.add('dragging');
        },
        
        onDragOver(event) {
            event.preventDefault();
        },
        
        onDrop(event, dropIndex, fieldType) {
            event.preventDefault();
            
            if (this.dragState.draggedFieldType !== fieldType) {
                return; // 只允许在同类型字段间拖拽
            }
            
            const draggedIndex = this.dragState.draggedIndex;
            if (draggedIndex === null || draggedIndex === dropIndex) {
                return;
            }
            
            // 执行字段重新排序
            const targetArray = this.selectedConfig[fieldType];
            const draggedItem = targetArray[draggedIndex];
            
            // 移除被拖拽的项
            targetArray.splice(draggedIndex, 1);
            
            // 在新位置插入
            const insertIndex = draggedIndex < dropIndex ? dropIndex - 1 : dropIndex;
            targetArray.splice(insertIndex, 0, draggedItem);
            
            // 清理拖拽状态
            this.dragState.draggedIndex = null;
            this.dragState.draggedFieldType = null;
            
            // 移除拖拽样式
            document.querySelectorAll('.dragging').forEach(el => {
                el.classList.remove('dragging');
            });
        },
        
        // 添加展示字段
        addDisplayField() {
            if (!this.selectedConfig.displayFields) {
                this.selectedConfig.displayFields = [];
            }
            this.selectedConfig.displayFields.push({
                field: '',
                label: '',
                width: null,
                sortable: false
            });
        },
        
        // 删除展示字段
        removeDisplayField(index) {
            if (this.selectedConfig.displayFields && index >= 0 && index < this.selectedConfig.displayFields.length) {
                this.selectedConfig.displayFields.splice(index, 1);
            }
        },
        
        // 添加搜索字段
        addSearchField() {
            if (!this.selectedConfig.searchFields) {
                this.selectedConfig.searchFields = [];
            }
            this.selectedConfig.searchFields.push({
                field: '',
                label: '',
                type: 'fuzzy',
                dict_source: '',
                dict_source_type: '' // 新增字典来源类型字段
            });
        },
        
        // 删除搜索字段
        removeSearchField(index) {
            if (this.selectedConfig.searchFields && index >= 0 && index < this.selectedConfig.searchFields.length) {
                this.selectedConfig.searchFields.splice(index, 1);
            }
        },
        
        // 处理字典来源类型变化
        onDictSourceTypeChange(field, index) {
            if (field.dict_source_type === 'custom') {
                // 切换到自定义模式，清空dict_source以便用户输入
                field.dict_source = '';
            } else if (field.dict_source_type === '') {
                // 选择无字典
                field.dict_source = '';
            } else {
                // 选择了字段名，将字段名设置为dict_source
                field.dict_source = field.dict_source_type;
            }
        }
    }
});

    // 添加Vue错误处理器
    app.config.errorHandler = (err, instance, info) => {
        console.error('Vue error handler caught:', err);
        console.error('Error info:', info);
        console.error('Component instance:', instance);
    };

    console.log('About to mount Vue app');
    console.log('App element after Vue creation:', appElement);

    try {
        app.mount('#app');
        console.log('Vue app mounted successfully');
    } catch (error) {
        console.error('Failed to mount Vue app:', error);
        console.error('Error stack:', error.stack);
    }
}); // 结束DOMContentLoaded事件监听器