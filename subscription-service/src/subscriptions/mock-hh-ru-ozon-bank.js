import { readFile } from "node:fs/promises";

const mockURL = new URL("./mock/hh-ru-ozon-bank-mock.json", import.meta.url);
const employerID = "2180";
const departmentID = "ozonru-2180-fintech";
const moscowID = "1";

const experienceFilter = new Map([
    ["noExperience", "none"],
    ["between1And3", "1-3"],
    ["between3And6", "3-6"],
    ["moreThan6", "6+"],
]);
const roleFilter = /\b(go|golang|backend|back-end|frontend|front-end)\b|бэкенд|бекенд|фронтенд|серверн|интерфейс/i;

function requireText(value, field) {
    if (typeof value !== "string" || !value.trim()) {
        throw new Error(`Missing or invalid ${field}`);
    }
    return value.normalize("NFKC").replace(/[\u0000-\u001f\u007f]/g, " ").replace(/\s+/g, " ").trim();
}

function requireUrl(value, field) {
    const str = requireText(value, field);
    const url = new URL(str);
    if (url.protocol !== "https:" || !url.hostname) {
        throw new Error(`Invalid HTTPS URL in ${field}`);
    }
    return url.href;
}

// Fetches the monopage from a mock dataset made to the JSON contract specifications of hh.ru
function loadAndValidatePage(data) {
    if (
        !data || !Array.isArray(data.items) ||
        data.page !== 0 || data.pages !== 1 ||
        !Number.isInteger(data.found) || data.found !== data.items.length ||
        !Number.isInteger(data.per_page) || data.per_page < data.items.length
    ) {
        throw new Error("Mock fixture is incomplete or malformed; refusing to send snapshot");
    }
    const seen = new Set();
    for (const item of data.items) {
        const id = requireText(item?.id, "id");
        if (seen.has(id)) throw new Error(`Duplicate vacancy id: ${id}`);
        seen.add(id);
        requireText(item.name, `name of ${id}`);
        requireUrl(item.alternate_url, `URL of ${id}`);
        requireText(item.employer?.id, `employer id of ${id}`);
        requireText(item.department?.id, `department id of ${id}`);
        requireText(item.area?.id, `area id of ${id}`);
        requireText(item.experience?.id, `experience id of ${id}`);
    }
    return data.items;
}

// Sends a mock api.hh.ru-like response without making any actual network requests
export async function updateSubscription({ all = false } = {}) {
    const data = JSON.parse(await readFile(mockURL, "utf8"));
    const items = loadAndValidatePage(data);

    const matching = items
        .filter(v => v.employer.id === employerID)
        .filter(v => v.department.id === departmentID)
        .filter(v => v.area.id === moscowID)
        .filter(v => experienceFilter.has(v.experience.id))
        .filter(v => roleFilter.test(v.name))
        .map(v => ({
            id: requireText(v.id, "id"),
            title: requireText(v.name, "title"),
            experience: experienceFilter.get(v.experience.id),
            url: requireUrl(v.alternate_url, "url"),
        }));

    return all ? matching : matching.filter(() => Math.random() < 0.10);
}
