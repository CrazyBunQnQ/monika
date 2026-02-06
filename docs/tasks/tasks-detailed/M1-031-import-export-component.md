# M1-031 实现 JSON 导入/导出组件

## 概述
实现角色卡 JSON 导入导出的 React 组件,支持文件选择、拖放上传、批量操作等。

## 验收标准
- [ ] 实现导出按钮和弹窗
- [ ] 实现导入文件选择
- [ ] 支持拖放上传
- [ ] 显示导入进度和结果
- [ ] 处理错误和警告
- [ ] 支持批量导入

## 技术方案

### 导出组件

```tsx
import React, { useState } from 'react';
import { Button } from '@/components/ui/Button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/Dialog';
import { Checkbox } from '@/components/ui/Checkbox';

interface ExportOptions {
  fields: string[];
  includeMetadata: boolean;
  format: 'json';
}

interface CharacterExportProps {
  characterId: string;
  characterName: string;
}

export const CharacterExport: React.FC<CharacterExportProps> = ({
  characterId,
  characterName
}) => {
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState<ExportOptions>({
    fields: [],
    includeMetadata: true,
    format: 'json'
  });
  const [exporting, setExporting] = useState(false);

  // 可导出字段
  const availableFields = [
    { value: 'id', label: 'ID' },
    { value: 'name', label: '名称' },
    { value: 'age', label: '年龄' },
    { value: 'occupation', label: '职业' },
    { value: 'player', label: '玩家' },
    { value: 'attributes', label: '属性' },
    { value: 'derived', label: '派生属性' },
    { value: 'skills', label: '技能' },
    { value: 'status', label: '状态' },
    { value: 'inventory', label: '物品' },
    { value: 'notes', label: '备注' },
  ];

  // 切换字段选择
  const toggleField = (field: string) => {
    setOptions(prev => ({
      ...prev,
      fields: prev.fields.includes(field)
        ? prev.fields.filter(f => f !== field)
        : [...prev.fields, field]
    }));
  };

  // 全选/取消全选
  const toggleAllFields = () => {
    if (options.fields.length === availableFields.length) {
      setOptions(prev => ({ ...prev, fields: [] }));
    } else {
      setOptions(prev => ({
        ...prev,
        fields: availableFields.map(f => f.value)
      }));
    }
  };

  // 执行导出
  const handleExport = async () => {
    setExporting(true);
    try {
      const params = new URLSearchParams();
      if (options.fields.length > 0) {
        params.append('fields', options.fields.join(','));
      }
      params.append('include_metadata', options.includeMetadata.toString());

      const response = await fetch(
        `/api/characters/${characterId}/export?${params}`
      );

      if (!response.ok) {
        throw new Error('导出失败');
      }

      // 获取文件名
      const contentDisposition = response.headers.get('Content-Disposition');
      const filenameMatch = /filename="(.+)"/.exec(contentDisposition || '');
      const filename = filenameMatch
        ? decodeURIComponent(filenameMatch[1])
        : `${characterName}.json`;

      // 下载文件
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);

      const a = document.createElement('a');
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      setOpen(false);
    } catch (error) {
      console.error('导出失败:', error);
      // 显示错误提示
    } finally {
      setExporting(false);
    }
  };

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => setOpen(true)}
      >
        <Download className="w-4 h-4 mr-2" />
        导出
      </Button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>导出角色卡</DialogTitle>
            <DialogDescription>
              选择要导出的字段和选项
            </DialogDescription>
          </DialogHeader>

          <div className="export-options">
            {/* 字段选择 */}
            <div className="option-section">
              <div className="option-header">
                <Checkbox
                  id="select-all"
                  checked={options.fields.length === availableFields.length}
                  onCheckedChange={toggleAllFields}
                />
                <Label htmlFor="select-all">全选字段</Label>
              </div>

              <div className="field-grid">
                {availableFields.map(field => (
                  <div key={field.value} className="field-item">
                    <Checkbox
                      id={`field-${field.value}`}
                      checked={options.fields.includes(field.value)}
                      onCheckedChange={() => toggleField(field.value)}
                    />
                    <Label htmlFor={`field-${field.value}`}>
                      {field.label}
                    </Label>
                  </div>
                ))}
              </div>
            </div>

            {/* 元数据选项 */}
            <div className="option-section">
              <div className="field-item">
                <Checkbox
                  id="include-metadata"
                  checked={options.includeMetadata}
                  onCheckedChange={(checked) =>
                    setOptions(prev => ({ ...prev, includeMetadata: !!checked }))
                  }
                />
                <Label htmlFor="include-metadata">
                  包含导出元数据
                </Label>
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setOpen(false)}
            >
              取消
            </Button>
            <Button
              onClick={handleExport}
              disabled={exporting || options.fields.length === 0}
            >
              {exporting ? '导出中...' : '导出'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
};
```

### 导入组件

```tsx
import React, { useState, useCallback } from 'react';
import { useDropzone } from 'react-dropzone';

interface ImportResult {
  total: number;
  success_count: number;
  failed_count: number;
  skipped_count: number;
  success: Array<{ id: string; name: string; action: string }>;
  failed: Array<{ id: string; reason: string }>;
  skipped: Array<{ id: string; reason: string }>;
}

interface CharacterImportProps {
  onImportComplete?: (result: ImportResult) => void;
}

export const CharacterImport: React.FC<CharacterImportProps> = ({
  onImportComplete
}) => {
  const [open, setOpen] = useState(false);
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<ImportResult | null>(null);
  const [resolveConflict, setResolveConflict] = useState<'error' | 'skip' | 'overwrite' | 'rename'>('error');

  // 文件拖放
  const onDrop = useCallback(async (acceptedFiles: File[]) => {
    const jsonFiles = acceptedFiles.filter(f => f.type === 'application/json' || f.name.endsWith('.json'));

    if (jsonFiles.length === 0) {
      // 显示错误: 无效文件
      return;
    }

    setImporting(true);
    setResult(null);

    try {
      const results: ImportResult[] = [];

      for (const file of jsonFiles) {
        const formData = new FormData();
        formData.append('file', file);
        formData.append('resolve_id_conflict', resolveConflict);

        const response = await fetch('/api/characters/import', {
          method: 'POST',
          body: formData
        });

        if (!response.ok) {
          throw new Error('导入失败');
        }

        const data = await response.json();
        results.push(data);
      }

      // 合并结果
      const merged: ImportResult = {
        total: results.reduce((sum, r) => sum + r.total, 0),
        success_count: results.reduce((sum, r) => sum + r.success_count, 0),
        failed_count: results.reduce((sum, r) => sum + r.failed_count, 0),
        skipped_count: results.reduce((sum, r) => sum + r.skipped_count, 0),
        success: results.flatMap(r => r.success),
        failed: results.flatMap(r => r.failed),
        skipped: results.flatMap(r => r.skipped),
      };

      setResult(merged);
      onImportComplete?.(merged);
    } catch (error) {
      console.error('导入失败:', error);
      // 显示错误
    } finally {
      setImporting(false);
    }
  }, [resolveConflict, onImportComplete]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'application/json': ['.json']
    },
    multiple: true
  });

  // 重置
  const handleReset = () => {
    setResult(null);
  };

  // 关闭
  const handleClose = () => {
    setOpen(false);
    setResult(null);
  };

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => setOpen(true)}
      >
        <Upload className="w-4 h-4 mr-2" />
        导入
      </Button>

      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>导入角色卡</DialogTitle>
            <DialogDescription>
              支持单个或批量导入 JSON 格式的角色卡文件
            </DialogDescription>
          </DialogHeader>

          {!result ? (
            <>
              {/* 冲突处理选项 */}
              <div className="conflict-options">
                <Label>ID 冲突处理方式:</Label>
                <div className="radio-group">
                  <label className="radio-item">
                    <input
                      type="radio"
                      value="error"
                      checked={resolveConflict === 'error'}
                      onChange={(e) => setResolveConflict(e.target.value)}
                    />
                    <span>报错(默认)</span>
                  </label>
                  <label className="radio-item">
                    <input
                      type="radio"
                      value="skip"
                      checked={resolveConflict === 'skip'}
                      onChange={(e) => setResolveConflict(e.target.value)}
                    />
                    <span>跳过</span>
                  </label>
                  <label className="radio-item">
                    <input
                      type="radio"
                      value="overwrite"
                      checked={resolveConflict === 'overwrite'}
                      onChange={(e) => setResolveConflict(e.target.value)}
                    />
                    <span>覆盖</span>
                  </label>
                  <label className="radio-item">
                    <input
                      type="radio"
                      value="rename"
                      checked={resolveConflict === 'rename'}
                      onChange={(e) => setResolveConflict(e.target.value)}
                    />
                    <span>重命名</span>
                  </label>
                </div>
              </div>

              {/* 拖放区域 */}
              <div
                {...getRootProps()}
                className={cn(
                  'dropzone',
                  isDragActive && 'dropzone--active'
                )}
              >
                <input {...getInputProps()} />
                <Upload className="dropzone-icon" />
                <p className="dropzone-text">
                  {isDragActive
                    ? '释放以上传文件'
                    : '拖放 JSON 文件到此处,或点击选择'}
                </p>
                <p className="dropzone-hint">
                  支持单个或多个文件
                </p>
              </div>

              {importing && (
                <div className="importing-state">
                  <div className="spinner" />
                  <p>导入中...</p>
                </div>
              )}
            </>
          ) : (
            <>
              {/* 导入结果 */}
              <div className="import-result">
                <div className="result-summary">
                  <h4>导入完成</h4>
                  <div className="stats">
                    <div className="stat">
                      <span className="stat-value">{result.total}</span>
                      <span className="stat-label">总计</span>
                    </div>
                    <div className="stat stat--success">
                      <span className="stat-value">{result.success_count}</span>
                      <span className="stat-label">成功</span>
                    </div>
                    <div className="stat stat--failed">
                      <span className="stat-value">{result.failed_count}</span>
                      <span className="stat-label">失败</span>
                    </div>
                    <div className="stat stat--skipped">
                      <span className="stat-value">{result.skipped_count}</span>
                      <span className="stat-label">跳过</span>
                    </div>
                  </div>
                </div>

                {/* 详细结果 */}
                {(result.failed.length > 0 || result.skipped.length > 0) && (
                  <div className="result-details">
                    {result.failed.length > 0 && (
                      <details>
                        <summary>失败 ({result.failed.length})</summary>
                        <ul>
                          {result.failed.map((item, idx) => (
                            <li key={idx}>
                              <strong>{item.id}</strong>: {item.reason}
                            </li>
                          ))}
                        </ul>
                      </details>
                    )}

                    {result.skipped.length > 0 && (
                      <details>
                        <summary>跳过 ({result.skipped.length})</summary>
                        <ul>
                          {result.skipped.map((item, idx) => (
                            <li key={idx}>
                              <strong>{item.id}</strong>: {item.reason}
                            </li>
                          ))}
                        </ul>
                      </details>
                    )}
                  </div>
                )}
              </div>

              <DialogFooter>
                <Button variant="outline" onClick={handleReset}>
                  继续导入
                </Button>
                <Button onClick={handleClose}>
                  完成
                </Button>
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
};
```

### 样式

```css
.export-options {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.option-section {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.option-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 0.5rem;
}

.field-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.dropzone {
  border: 2px dashed #cbd5e1;
  border-radius: 8px;
  padding: 3rem 1.5rem;
  text-align: center;
  cursor: pointer;
  transition: all 0.2s;
}

.dropzone--active {
  border-color: #5c6bc0;
  background: #f5f7ff;
}

.dropzone-icon {
  width: 48px;
  height: 48px;
  margin: 0 auto 1rem;
  color: #6c757d;
}

.dropzone-text {
  font-weight: 600;
  margin-bottom: 0.25rem;
}

.dropzone-hint {
  font-size: 0.875rem;
  color: #6c757d;
}

.conflict-options {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.radio-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.radio-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
}

.importing-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 2rem;
  gap: 1rem;
}

.import-result {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.result-summary {
  padding: 1rem;
  background: #f8f9fa;
  border-radius: 8px;
}

.result-summary h4 {
  margin-bottom: 0.75rem;
}

.stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.75rem;
}

.stat {
  text-align: center;
}

.stat-value {
  display: block;
  font-size: 1.5rem;
  font-weight: 700;
}

.stat-label {
  font-size: 0.875rem;
  color: #6c757d;
}

.stat--success .stat-value {
  color: #66bb6a;
}

.stat--failed .stat-value {
  color: #ef5350;
}

.stat--skipped .stat-value {
  color: #ffa726;
}

.result-details details {
  margin-top: 0.5rem;
}

.result-details summary {
  cursor: pointer;
  font-weight: 600;
}

.result-details ul {
  margin-top: 0.5rem;
  padding-left: 1.5rem;
}

.result-details li {
  margin-bottom: 0.25rem;
  font-size: 0.875rem;
}
```

## 依赖关系
- 前置任务: M1-026 实现角色卡导出 JSON, M1-027 实现角色卡导入
- 被依赖: M1-028 实现角色卡列表组件

## 预估工时
1h
