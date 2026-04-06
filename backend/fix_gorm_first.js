import fs from 'fs';
import path from 'path';

function fixGormFirst(dir) {
    const files = fs.readdirSync(dir);
    for (const file of files) {
        const fullPath = path.join(dir, file);
        if (fs.statSync(fullPath).isDirectory()) {
            fixGormFirst(fullPath);
        } else if (fullPath.endsWith('.go')) {
            let content = fs.readFileSync(fullPath, 'utf8');
            let updated = false;

            // Pattern: First(&variable, id) -> First(&variable, "id = ?", id)
            // Need to carefully handle Select("...").First(&var, id) as well.
            // A regex to match .First(&[a-zA-Z0-9_.]+,\s*[a-zA-Z0-9_.()]+)
            // But we must NOT match if there is already a condition string like First(&var, "id = ?", id)
            
            // Regex explanation:
            // \.First\(&([a-zA-Z0-9_.\[\]]+),\s*([a-zA-Z0-9_.\[\]()]+)\)
            // It will capture `project` and `input.ProjectID`
            
            const regex = /\.First\(&([a-zA-Z0-9_.\[\]]+),\s*([^",)]+)\)/g;
            
            const newContent = content.replace(regex, (match, p1, p2) => {
                updated = true;
                return `.First(&${p1}, "id = ?", ${p2.trim()})`;
            });

            if (updated) {
                fs.writeFileSync(fullPath, newContent, 'utf8');
                console.log(`Updated ${fullPath}`);
            }
        }
    }
}

fixGormFirst('d:/T-Monitor/backend/internal/handlers');
