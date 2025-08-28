const { createApp } = Vue;

// 创建专用的axios实例
// 全局配置管理器（和 app.js 保持一致）
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

const crudAxios = axios.create();

createApp({
    data() {
        return {
            configName: '',
            loading: true,
            records: [],
            searchFields: [],
            sortableFields: [],
            displayFields: [],
            updatableFields: [],
            creatableFields: [],
            tableFields: [],
            editableFields: [],
            dictData: {},
            parsedSqlFields: [], // 添加这个来存储解析的SQL字段
            filters: {},
            currentSort: '',
            sortOrder: 'asc',
            currentPage: 1,
            pageSize: 20,
            totalRecords: 0,
            totalPages: 0,
            editingRecord: null,
            formData: {},
            saving: false,
            modal: null
        }
    },
    computed: {
        paginationPages() {
            const pages = [];
            const start = Math.max(1, this.currentPage - 2);
            const end = Math.min(this.totalPages, this.currentPage + 2);
            
            for (let i = start; i <= end; i++) {
                pages.push(i);
            }
            return pages;
        }
    },
    async mounted() {
        // 首先加载配置
        ConfigManager.loadConfig();
        
        // 从URL获取配置名称
        const pathParts = window.location.pathname.split('/');
        this.configName = pathParts[pathParts.length - 1];
        
        // 初始化Bootstrap模态框
        this.modal = new bootstrap.Modal(document.getElementById('recordModal'));
        
        await this.loadConfiguration();
        await this.loadData();
        this.loading = false;
    },
    methods: {
        
        async loadConfiguration() {
            try {
                // 直接根据配置名称获取单个配置，而不是获取所有配置再筛选
                const response = await crudAxios.get(ConfigManager.getApiUrl(`/configs/by-name/${this.configName}`));
                const config = response.data.data;
                
                if (!config) {
                    throw new Error(`Configuration '${this.configName}' not found`);
                }
                
                // 解析展示字段
                if (config.query_display_fields) {
                    this.displayFields = JSON.parse(config.query_display_fields);
                }
                
                // 解析搜索字段
                if (config.query_search_fields) {
                    this.searchFields = JSON.parse(config.query_search_fields);
                }
                
                // 解析排序字段
                if (config.query_sortable_fields) {
                    this.sortableFields = JSON.parse(config.query_sortable_fields);
                }
                
                // 解析可更新字段
                if (config.update_updatable_fields) {
                    this.updatableFields = JSON.parse(config.update_updatable_fields);
                }
                
                // 解析可创建字段
                if (config.create_creatable_fields) {
                    this.creatableFields = JSON.parse(config.create_creatable_fields);
                }
                
                // 如果配置为空，尝试从SQL语句解析字段
                const sqlFields = this.parseSQLFields(config.create_statement);
                this.parsedSqlFields = sqlFields; // 保存解析的字段
                console.log('Parsed SQL fields:', sqlFields);
                
                // 如果没有配置展示字段，使用SQL解析的所有字段
                if (this.displayFields.length === 0 && sqlFields.length > 0) {
                    this.displayFields = sqlFields.map(field => ({
                        field: field.name,
                        label: field.name,
                        width: null,
                        sortable: field.name !== 'id'
                    }));
                    console.log('Set display fields:', this.displayFields);
                }
                
                // 如果没有配置可更新字段，使用SQL解析的字段（排除id）
                if (this.updatableFields.length === 0 && sqlFields.length > 0) {
                    this.updatableFields = sqlFields.filter(field => field.name !== 'id').map(field => field.name);
                    console.log('Set updatable fields:', this.updatableFields);
                }
                
                // 如果没有配置排序字段，使用SQL解析的所有字段
                if (this.sortableFields.length === 0 && sqlFields.length > 0) {
                    this.sortableFields = sqlFields.map(field => field.name);
                    console.log('Set sortable fields:', this.sortableFields);
                }
                
                // 如果没有配置搜索字段，为文本和数字字段创建默认搜索配置
                if (this.searchFields.length === 0 && sqlFields.length > 0) {
                    this.searchFields = sqlFields.map(field => {
                        const searchType = this.getDefaultSearchType(field.type);
                        return {
                            field: field.name,
                            type: searchType,
                            dict_source: null
                        };
                    });
                    console.log('Set search fields:', this.searchFields);
                }
                
                // 加载字典数据
                await this.loadDictData();
                
                // 初始化多选字段的过滤器为空数组
                this.searchFields.forEach(field => {
                    if (field.type === 'multi_select') {
                        this.filters[field.field] = [];
                    }
                });
                
            } catch (error) {
                console.error('Failed to load configuration:', error);
                alert('配置加载失败: ' + error.message);
            }
        },
        
        // 从SQL类型推断搜索类型
        getDefaultSearchType(sqlType) {
            const type = sqlType.toLowerCase();
            if (type.includes('varchar') || type.includes('text') || type.includes('char')) {
                return 'fuzzy';
            } else if (type.includes('int') || type.includes('numeric') || type.includes('decimal') || type.includes('float') || type.includes('double')) {
                return 'range';
            } else {
                return 'exact';
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
                    if (!trimmed || trimmed.toLowerCase().includes('constraint') || 
                        trimmed.toLowerCase().includes('primary key') ||
                        trimmed.toLowerCase().includes('foreign key') ||
                        trimmed.toLowerCase().includes('unique') ||
                        trimmed.toLowerCase().includes('check')) {
                        return;
                    }
                    
                    // 匹配字段名和类型
                    const fieldMatch = trimmed.match(/^(\w+)\s+(\w+(?:\([^)]*\))?)/i);
                    if (fieldMatch) {
                        fields.push({
                            name: fieldMatch[1].toLowerCase(),
                            type: fieldMatch[2].toLowerCase()
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
        
        async loadDictData() {
            for (const field of this.searchFields) {
                if (field.dict_source && (field.type === 'single' || field.type === 'multi_select')) {
                    try {
                        const response = await crudAxios.get(ConfigManager.getApiUrl(`/${this.configName}/dict/${field.field}`));
                        this.dictData[field.field] = response.data.data;
                    } catch (error) {
                        console.warn(`Failed to load dict data for ${field.field}:`, error);
                    }
                }
            }
        },
        
        async loadData() {
            try {
                this.loading = true;
                const params = new URLSearchParams();
                
                // 分页参数
                params.append('page', this.currentPage);
                params.append('page_size', this.pageSize);
                
                // 搜索参数
                Object.keys(this.filters).forEach(key => {
                    if (this.filters[key] !== undefined && this.filters[key] !== '') {
                        if (key.endsWith('_min') || key.endsWith('_max')) {
                            // 范围搜索处理（数字范围）
                            const baseField = key.replace(/_min$/, '').replace(/_max$/, '');
                            const minValue = this.filters[baseField + '_min'];
                            const maxValue = this.filters[baseField + '_max'];
                            
                            if (minValue !== undefined && minValue !== '') {
                                params.append(baseField, JSON.stringify({min: minValue, max: maxValue}));
                            }
                        } else if (key.endsWith('_start') || key.endsWith('_end')) {
                            // 日期范围搜索处理
                            const baseField = key.replace(/_start$/, '').replace(/_end$/, '');
                            const startValue = this.filters[baseField + '_start'];
                            const endValue = this.filters[baseField + '_end'];
                            
                            if (startValue !== undefined && startValue !== '') {
                                // 将日期转换为时间戳
                                const startTimestamp = startValue ? new Date(startValue).getTime() / 1000 : null;
                                const endTimestamp = endValue ? new Date(endValue + 'T23:59:59').getTime() / 1000 : null;
                                
                                const rangeData = {};
                                if (startTimestamp) rangeData.start = startTimestamp;
                                if (endTimestamp) rangeData.end = endTimestamp;
                                
                                if (Object.keys(rangeData).length > 0) {
                                    params.append(baseField, JSON.stringify(rangeData));
                                }
                            }
                        } else if (!key.endsWith('_min') && !key.endsWith('_max') && !key.endsWith('_start') && !key.endsWith('_end')) {
                            // 检查是否是多选字段
                            const searchField = this.searchFields.find(f => f.field === key);
                            if (searchField && searchField.type === 'multi_select') {
                                // 多选字段：值应该是数组，转换为JSON字符串发送
                                if (Array.isArray(this.filters[key]) && this.filters[key].length > 0) {
                                    params.append(key, JSON.stringify(this.filters[key]));
                                }
                            } else {
                                // 普通字段
                                params.append(key, this.filters[key]);
                            }
                        }
                    }
                });
                
                // 排序参数
                if (this.currentSort) {
                    params.append('sort', this.currentSort);
                    params.append('order', this.sortOrder);
                }
                
                const response = await crudAxios.get(ConfigManager.getApiUrl(`/${this.configName}/list?${params.toString()}`));
                const result = response.data.data;
                
                this.records = result.data || [];
                this.totalRecords = result.total;
                this.totalPages = result.total_pages;
                this.currentPage = result.page;
                
                // 提取表格字段 - 优先使用展示字段配置的顺序
                if (this.displayFields.length > 0) {
                    // 使用配置的展示字段顺序
                    this.tableFields = this.displayFields.map(field => field.field);
                } else if (this.records.length > 0) {
                    this.tableFields = Object.keys(this.records[0]);
                } else if (this.parsedSqlFields && this.parsedSqlFields.length > 0) {
                    // 如果没有记录，从SQL解析的字段中获取
                    this.tableFields = this.parsedSqlFields.map(field => field.name);
                }
                
                // 设置可编辑字段 - 优先使用可创建字段配置
                if (this.creatableFields.length > 0) {
                    // 如果配置了可创建字段，使用配置的字段并保持顺序
                    // 过滤掉user_readonly为true的字段（不可编辑的字段）
                    this.editableFields = this.creatableFields.filter(field => !field.user_readonly);
                } else if (this.updatableFields.length > 0) {
                    // 如果配置了可更新字段，使用配置的字段
                    this.editableFields = this.updatableFields;
                } else if (this.tableFields.length > 0) {
                    // 否则使用所有字段除了id
                    this.editableFields = this.tableFields.filter(field => field !== 'id').map(field => ({ field: field, label: field, type: 'text' }));
                } else if (this.parsedSqlFields && this.parsedSqlFields.length > 0) {
                    // 最后从SQL解析字段中获取（排除id）
                    this.editableFields = this.parsedSqlFields.filter(field => field.name !== 'id').map(field => ({ field: field.name, label: field.name, type: 'text' }));
                }
                
                console.log('Table fields:', this.tableFields);
                console.log('Editable fields:', this.editableFields);
                
            } catch (error) {
                console.error('Failed to load data:', error);
                alert('数据加载失败: ' + error.message);
            } finally {
                this.loading = false;
            }
        },
        
        async applyFilters() {
            this.currentPage = 1;
            await this.loadData();
        },
        
        clearFilters() {
            // 重新初始化filters对象，对多选字段设置为空数组
            this.filters = {};
            this.searchFields.forEach(field => {
                if (field.type === 'multi_select') {
                    this.filters[field.field] = [];
                }
            });
            this.currentPage = 1;
            this.loadData();
        },
        
        toggleSort(field) {
            if (this.currentSort === field) {
                this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc';
            } else {
                this.currentSort = field;
                this.sortOrder = 'asc';
            }
            this.loadData();
        },
        
        getSortIcon(field) {
            if (this.currentSort !== field) {
                return 'bi-arrow-down-up';
            }
            return this.sortOrder === 'asc' ? 'bi-sort-up' : 'bi-sort-down';
        },
        
        async changePage(page) {
            if (page >= 1 && page <= this.totalPages && page !== this.currentPage) {
                this.currentPage = page;
                await this.loadData();
            }
        },
        
        refreshData() {
            this.loadData();
        },
        
        showCreateModal() {
            this.editingRecord = null;
            this.formData = {};
            // 为每个可编辑字段初始化空值
            this.editableFields.forEach(field => {
                const fieldName = typeof field === 'string' ? field : field.field;
                this.formData[fieldName] = '';
            });
            this.modal.show();
        },
        
        editRecord(record) {
            this.editingRecord = record;
            this.formData = { ...record };
            this.modal.show();
        },
        
        async saveRecord() {
            try {
                this.saving = true;
                
                if (this.editingRecord) {
                    // 更新记录
                    const id = this.editingRecord.id;
                    await crudAxios.put(ConfigManager.getApiUrl(`/${this.configName}/update/${id}`), this.formData);
                } else {
                    // 创建记录
                    await crudAxios.post(ConfigManager.getApiUrl(`/${this.configName}/create`), this.formData);
                }
                
                this.modal.hide();
                await this.loadData();
                
            } catch (error) {
                console.error('Failed to save record:', error);
                alert('保存失败: ' + (error.response?.data?.error || error.message));
            } finally {
                this.saving = false;
            }
        },
        
        async deleteRecord(record) {
            if (!confirm(`确定要删除这条记录吗？`)) {
                return;
            }
            
            try {
                const id = record.id;
                await crudAxios.delete(ConfigManager.getApiUrl(`/${this.configName}/delete/${id}`));
                await this.loadData();
            } catch (error) {
                console.error('Failed to delete record:', error);
                alert('删除失败: ' + (error.response?.data?.error || error.message));
            }
        },
        
        formatValue(value) {
            if (value === null || value === undefined) {
                return '-';
            }
            if (typeof value === 'object') {
                return JSON.stringify(value);
            }
            return String(value);
        },
        
        // 获取字段标签
        getFieldLabel(field) {
            if (typeof field === 'string') {
                return field;
            }
            return field.label || field.field || field;
        },
        
        // 获取字段名
        getFieldName(field) {
            if (typeof field === 'string') {
                return field;
            }
            return field.field || field;
        },
        
        // 获取字段类型
        getFieldType(field) {
            if (typeof field === 'string') {
                return 'text';
            }
            return field.type || 'text';
        },
        
        // 根据值获取标签（用于多选显示）
        getLabelByValue(fieldName, value) {
            if (!this.dictData[fieldName]) return value;
            const item = this.dictData[fieldName].find(item => item.value === value);
            return item ? item.label : value;
        },
        
        // 清空多选字段
        clearMultiSelect(fieldName) {
            this.filters[fieldName] = [];
        }
    }
}).mount('#app');