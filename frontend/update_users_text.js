import fs from 'fs';
let content = fs.readFileSync('src/routes/dashboard/users/+page.svelte', 'utf-8');

content = content.replace(
    "{u.role === 'admin' ? 'Demote to User' : 'Promote to Admin'}",
    "{u.role === 'admin' ? 'ปรับเป็น User (Demote)' : 'ตั้งเป็น Admin (Promote)'}"
);

content = content.replace(
    "Approve\n                    </button>",
    "ยืนยัน (Approve)\n                    </button>"
);

fs.writeFileSync('src/routes/dashboard/users/+page.svelte', content);
console.log("Updated users page text successfully");
