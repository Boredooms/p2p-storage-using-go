'use client';

import { Editor } from '@monaco-editor/react';
import { useState } from 'react';

interface CodeEditorProps {
    language: string;
    defaultValue?: string;
    onChange?: (value: string | undefined) => void;
}

export default function CodeEditor({ language, defaultValue, onChange }: CodeEditorProps) {
    const [theme, setTheme] = useState<'vs-dark' | 'light'>('vs-dark');

    return (
        <div className="border border-gray-700 rounded-lg overflow-hidden">
            <div className="bg-gray-800 px-4 py-2 flex justify-between items-center">
                <span className="text-sm text-gray-300">Code Editor - {language}</span>
                <button
                    onClick={() => setTheme(theme === 'vs-dark' ? 'light' : 'vs-dark')}
                    className="text-xs px-3 py-1 bg-gray-700 hover:bg-gray-600 rounded transition-colors"
                >
                    Toggle Theme
                </button>
            </div>
            <Editor
                height="400px"
                language={language}
                theme={theme}
                defaultValue={defaultValue}
                onChange={onChange}
                options={{
                    minimap: { enabled: false },
                    fontSize: 14,
                    lineNumbers: 'on',
                    roundedSelection: false,
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                }}
            />
        </div>
    );
}
