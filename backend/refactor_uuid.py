import os
import re

handlers_dir = r"d:\T-Monitor\backend\internal\handlers"
middleware_dir = r"d:\T-Monitor\backend\internal\middleware"

def refactor_file(filepath):
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    # 1. Replace struct field types that use uint for IDs
    # e.g. ProjectID uint -> ProjectID uuid.UUID
    content = re.sub(r'([A-Za-z]+ID\s+)uint', r'\1uuid.UUID', content)
    content = re.sub(r'([A-Za-z]+ID\s+)\*uint', r'\1*uuid.UUID', content)
    content = re.sub(r'(ID\s+)uint', r'\1uuid.UUID', content)

    # 2. Replace type assertions for c.Locals("user_id").(uint) -> c.Locals("user_id").(string)
    # Actually wait, let's cast c.Locals("user_id") to string, and then if needed, just use it as string
    # Because if JWT subject provides it, it will be a string. 
    # But wait, UserID in models is uuid.UUID. So c.Locals("user_id") should be type uuid.UUID.
    # Where does it get set? In middleware. 
    content = content.replace('.(uint)', '.(uuid.UUID)')

    # 3. Replace QueryInt with Query for project_id
    content = re.sub(r'c\.QueryInt\("([^"]+)"\)', r'c.Query("\1")', content)

    # 4. uint(projectID) casting -> we should remove this.
    # if projectID is retrieved via Query, it's a string, so we need to parse it or use uuid.Parse
    # Wait, the best way for struct initialization is uuid.MustParse() or just rely on the fact that if it's string, we can't assign to uuid.UUID.
    
    # 5. Make sure github.com/google/uuid is imported
    if 'uuid.UUID' in content and 'github.com/google/uuid' not in content:
        content = re.sub(r'import \(', 'import (\n\t"github.com/google/uuid"\n', content, count=1)

    with open(filepath, 'w', encoding='utf-8') as f:
        f.write(content)

for filename in os.listdir(handlers_dir):
    if filename.endswith(".go"):
        refactor_file(os.path.join(handlers_dir, filename))

for filename in os.listdir(middleware_dir):
    if filename.endswith(".go"):
        refactor_file(os.path.join(middleware_dir, filename))
