import fs from 'fs';
import path from 'path';

function searchFiles(dir) {
    let results = [];
    const files = fs.readdirSync(dir);
    for (const file of files) {
        const fullPath = path.join(dir, file);
        if (fs.statSync(fullPath).isDirectory()) {
            results = results.concat(searchFiles(fullPath));
        } else if (fullPath.endsWith('.svelte') || fullPath.endsWith('.ts')) {
            const content = fs.readFileSync(fullPath, 'utf8');
            const lines = content.split('\n');
            lines.forEach((line, index) => {
                if (line.includes('Number(') || line.includes('parseInt(')) {
                    if (line.toLowerCase().includes('id')) {
                        results.push(`${fullPath}:${index + 1}: ${line.trim()}`);
                    }
                }
                if (line.match(/id:\s*number/i) || line.match(/Id:\s*number/)) {
                    results.push(`${fullPath}:${index + 1} (TYPE:NUMBER): ${line.trim()}`);
                }
            });
        }
    }
    return results;
}

const res = searchFiles('d:/T-Monitor/frontend/src');
fs.writeFileSync('d:/T-Monitor/frontend/search_results.utf8.txt', res.join('\n'), 'utf8');
