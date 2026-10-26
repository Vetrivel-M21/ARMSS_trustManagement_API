package landing

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RenderLandingPage serves the official Nesakkaram Charitable Trust public website
// on the root route ("/") of the server, fully equipped with trust background,
// beneficiary wings, and the mandatory compliance policies required by Payment Gateways (Cashfree / RBI).
func RenderLandingPage(c *gin.Context) {
	const html = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Nesakkaram Charitable Trust | Official Trust Portal</title>
    <meta name="description" content="Nesakkaram Charitable Trust - Dedicated to the care, shelter, and empowerment of children, special-needs individuals, and senior citizens across our three welfare branches.">
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Nunito:wght@400;600;700;800&family=Outfit:wght@500;700;800;900&display=swap" rel="stylesheet">
    <style>
        :root {
            --primary: #059669;
            --primary-dark: #047857;
            --primary-light: #ecfdf5;
            --slate-900: #0f172a;
            --slate-800: #1e293b;
            --slate-700: #334155;
            --slate-600: #475569;
            --slate-500: #64748b;
            --slate-100: #f1f5f9;
            --slate-50: #f8fafc;
        }
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: 'Nunito', -apple-system, sans-serif;
            background-color: var(--slate-50);
            color: var(--slate-800);
            line-height: 1.6;
        }
        h1, h2, h3, h4, .brand-font { font-family: 'Outfit', sans-serif; }
        a { color: var(--primary); text-decoration: none; }
        a:hover { text-decoration: underline; }

        /* Navigation Bar */
        .navbar {
            background: #ffffff;
            border-bottom: 1px solid #e2e8f0;
            position: sticky;
            top: 0;
            z-index: 50;
            box-shadow: 0 1px 3px rgba(0,0,0,0.05);
        }
        .nav-container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 14px 20px;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        .brand {
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .brand-badge {
            width: 44px;
            height: 44px;
            background: var(--primary);
            color: #ffffff;
            border-radius: 10px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 22px;
            font-weight: 800;
            box-shadow: 0 4px 6px -1px rgba(5, 150, 105, 0.3);
        }
        .brand-name {
            font-size: 18px;
            font-weight: 800;
            color: var(--slate-900);
            letter-spacing: -0.3px;
        }
        .brand-sub {
            font-size: 11px;
            font-weight: 600;
            color: var(--primary);
            text-transform: uppercase;
            letter-spacing: 0.6px;
        }
        .nav-links {
            display: flex;
            align-items: center;
            gap: 24px;
            list-style: none;
        }
        .nav-links a {
            font-size: 14px;
            font-weight: 700;
            color: var(--slate-700);
            transition: color 0.2s;
        }
        .nav-links a:hover {
            color: var(--primary);
            text-decoration: none;
        }
        .btn-donate {
            background: var(--primary);
            color: #ffffff !important;
            padding: 9px 18px;
            border-radius: 8px;
            font-weight: 700;
            font-size: 14px;
            transition: background 0.2s, transform 0.1s;
            box-shadow: 0 2px 4px rgba(5, 150, 105, 0.2);
        }
        .btn-donate:hover {
            background: var(--primary-dark);
            transform: translateY(-1px);
        }

        /* Hero Section */
        .hero {
            background: linear-gradient(135deg, #064e3b 0%, #065f46 50%, #047857 100%);
            color: #ffffff;
            padding: 60px 20px 80px;
            text-align: center;
            position: relative;
        }
        .hero-container {
            max-width: 900px;
            margin: 0 auto;
        }
        .hero-pill {
            display: inline-flex;
            align-items: center;
            gap: 8px;
            background: rgba(255,255,255,0.15);
            padding: 6px 16px;
            border-radius: 30px;
            font-size: 12px;
            font-weight: 700;
            letter-spacing: 0.5px;
            margin-bottom: 20px;
            backdrop-filter: blur(4px);
            border: 1px solid rgba(255,255,255,0.2);
        }
        .hero h1 {
            font-size: 40px;
            font-weight: 900;
            line-height: 1.2;
            margin-bottom: 16px;
            letter-spacing: -0.5px;
        }
        .hero p {
            font-size: 17px;
            color: #d1fae5;
            margin-bottom: 30px;
            line-height: 1.6;
        }
        .badges-row {
            display: flex;
            justify-content: center;
            flex-wrap: wrap;
            gap: 16px;
        }
        .badge-chip {
            background: rgba(255,255,255,0.1);
            border: 1px solid rgba(255,255,255,0.25);
            padding: 8px 16px;
            border-radius: 8px;
            font-size: 13px;
            font-weight: 700;
        }

        /* Container & Cards */
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 60px 20px;
        }
        .section-title {
            text-align: center;
            margin-bottom: 40px;
        }
        .section-title h2 {
            font-size: 28px;
            font-weight: 800;
            color: var(--slate-900);
            margin-bottom: 8px;
        }
        .section-title p {
            color: var(--slate-500);
            font-size: 15px;
        }

        .grid-3 {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
            gap: 24px;
        }
        .card {
            background: #ffffff;
            border: 1px solid #e2e8f0;
            border-radius: 14px;
            padding: 26px;
            box-shadow: 0 4px 6px -1px rgba(0,0,0,0.03);
            transition: transform 0.2s, box-shadow 0.2s;
        }
        .card:hover {
            transform: translateY(-3px);
            box-shadow: 0 10px 15px -3px rgba(0,0,0,0.06);
        }
        .card-icon {
            width: 48px;
            height: 48px;
            background: var(--primary-light);
            color: var(--primary);
            border-radius: 12px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 24px;
            margin-bottom: 16px;
        }
        .card h3 {
            font-size: 18px;
            font-weight: 800;
            color: var(--slate-900);
            margin-bottom: 8px;
        }
        .card p {
            font-size: 14px;
            color: var(--slate-600);
            line-height: 1.6;
        }

        /* Policy & Compliance Tabs */
        .policy-box {
            background: #ffffff;
            border: 1px solid #e2e8f0;
            border-radius: 14px;
            padding: 32px;
            margin-bottom: 30px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.02);
        }
        .policy-box h3 {
            font-size: 20px;
            font-weight: 800;
            color: var(--slate-900);
            margin-bottom: 14px;
            display: flex;
            align-items: center;
            gap: 10px;
        }
        .policy-box h3::before {
            content: '';
            display: inline-block;
            width: 6px;
            height: 20px;
            background: var(--primary);
            border-radius: 4px;
        }
        .policy-box p, .policy-box li {
            font-size: 14px;
            color: var(--slate-600);
            line-height: 1.7;
            margin-bottom: 10px;
        }
        .policy-box ul {
            padding-left: 20px;
            margin-bottom: 12px;
        }

        /* Contact Details Grid */
        .contact-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
            gap: 20px;
        }
        .contact-card {
            background: var(--slate-100);
            border: 1px solid #e2e8f0;
            border-radius: 12px;
            padding: 20px;
        }
        .contact-card h4 {
            font-size: 14px;
            font-weight: 800;
            color: var(--slate-700);
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 6px;
        }
        .contact-card p {
            font-size: 15px;
            font-weight: 700;
            color: var(--slate-900);
        }

        /* Footer */
        footer {
            background: var(--slate-900);
            color: #94a3b8;
            padding: 40px 20px 30px;
            font-size: 13px;
        }
        .footer-content {
            max-width: 1200px;
            margin: 0 auto;
            display: flex;
            flex-direction: column;
            gap: 20px;
            align-items: center;
            text-align: center;
        }
        .footer-links {
            display: flex;
            flex-wrap: wrap;
            gap: 20px;
            justify-content: center;
        }
        .footer-links a {
            color: #cbd5e1;
            font-weight: 600;
        }
        .footer-links a:hover {
            color: #ffffff;
        }

        @media (max-width: 768px) {
            .nav-links { display: none; }
            .hero h1 { font-size: 28px; }
            .hero p { font-size: 15px; }
            .policy-box { padding: 20px; }
        }
    </style>
</head>
<body>

    <!-- Header Navigation -->
    <header class="navbar">
        <div class="nav-container">
            <div class="brand">
                <div class="brand-badge">
                    <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                        <path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z"/>
                    </svg>
                </div>
                <div>
                    <div class="brand-name">NESAKKARAM CHARITABLE TRUST</div>
                    <div class="brand-sub">Registered Charitable Trust &bull; Serving Humanity</div>
                </div>
            </div>
            <nav>
                <ul class="nav-links">
                    <li><a href="#about">About Us</a></li>
                    <li><a href="#wings">Our Branches</a></li>
                    <li><a href="#policies">Policies</a></li>
                    <li><a href="#contact">Contact</a></li>
                    <li><a href="#contact" class="btn-donate">Donate Online</a></li>
                </ul>
            </nav>
        </div>
    </header>

    <!-- Hero Section -->
    <section class="hero">
        <div class="hero-container">
            <div class="hero-pill">
                <span>Registered Charitable Trust &bull; Dedicated to Welfare &amp; Care</span>
            </div>
            <h1>Serving Humanity with Dignity, Care &amp; Compassion</h1>
            <p>Nesakkaram Charitable Trust provides shelter, nutritious daily meals, healthcare, and educational empowerment across our three dedicated welfare branches.</p>
            <div class="badges-row" id="heroBranches">
                <div class="badge-chip">Old Age Home</div>
                <div class="badge-chip">Children Home</div>
                <div class="badge-chip">Children Adoption Home</div>
            </div>
        </div>
    </section>

    <!-- Main Content -->
    <main class="container">

        <!-- About Us Section -->
        <section id="about" style="margin-bottom: 60px;">
            <div class="section-title">
                <h2>About Nesakkaram Charitable Trust</h2>
                <p>Committed to uplifting the underprivileged through direct humanitarian service</p>
            </div>
            <div class="card" style="line-height: 1.8; font-size: 15px;">
                <p style="margin-bottom: 14px;"><strong>Nesakkaram Charitable Trust</strong> is a legally constituted, recognized non-profit charitable trust established in Tamil Nadu, India. The trust was founded with the singular vision of extending a helping hand ("Nesakkaram") to vulnerable children, individuals with special needs, and destitute senior citizens who require lifelong care, dignity, and family love.</p>
                <p>Our institutions operate 365 days a year across our three dedicated branches, providing safe lodging, medical supervision, psychological support, wholesome nutritional diets (Annadhanam), and formal schooling to nurture every resident into a confident, self-reliant individual. All contributions are audited annually and managed with strict financial transparency.</p>
            </div>
        </section>

        <!-- Our 3 Branches -->
        <section id="wings" style="margin-bottom: 60px;">
            <div class="section-title">
                <h2>Our 3 Operational Branches</h2>
                <p>Dedicated facilities operating round-the-clock for humanitarian welfare</p>
            </div>
            <div class="grid-3" id="branchesContainer">
                <div class="card">
                    <span class="live-tag" style="display:inline-block;background:#dcfce7;color:#15803d;font-size:11px;font-weight:700;padding:3px 8px;border-radius:6px;margin-bottom:10px;">Branch 1</span>
                    <div class="card-icon" style="width:48px;height:48px;background:var(--primary-light);color:var(--primary);border-radius:12px;display:flex;align-items:center;justify-content:center;margin-bottom:16px;">
                        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.3 1.5 4.05 3 5.5l7 7Z"/></svg>
                    </div>
                    <h3>Old Age Home</h3>
                    <div style="font-size:14px;font-weight:700;color:var(--primary);margin-bottom:10px;">&#2990;&#3009;&#2980;&#3007;&#2991;&#3019;&#2992;&#3021;&#2965;&#2995;&#3021; &#2951;&#2994;&#3021;&#2994;&#2990;&#3021;</div>
                    <p>Providing peaceful, loving shelter, palliative medical support, wholesome meals, and companionship to senior citizens and destitute elders.</p>
                </div>
                <div class="card">
                    <span class="live-tag" style="display:inline-block;background:#dcfce7;color:#15803d;font-size:11px;font-weight:700;padding:3px 8px;border-radius:6px;margin-bottom:10px;">Branch 2</span>
                    <div class="card-icon" style="width:48px;height:48px;background:var(--primary-light);color:var(--primary);border-radius:12px;display:flex;align-items:center;justify-content:center;margin-bottom:16px;">
                        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
                    </div>
                    <h3>Children Home</h3>
                    <div style="font-size:14px;font-weight:700;color:var(--primary);margin-bottom:10px;">&#2965;&#3009;&#2996;&#2984;&#3021;&#2980;&#3016;&#2965;&#2994;&#3021; &#2951;&#2994;&#3021;&#2994;&#2990;&#3021;</div>
                    <p>Safe residential shelter offering comprehensive education, nutritious daily meals, health monitoring, and life skills for underprivileged and orphan children.</p>
                </div>
                <div class="card">
                    <span class="live-tag" style="display:inline-block;background:#dcfce7;color:#15803d;font-size:11px;font-weight:700;padding:3px 8px;border-radius:6px;margin-bottom:10px;">Branch 3</span>
                    <div class="card-icon" style="width:48px;height:48px;background:var(--primary-light);color:var(--primary);border-radius:12px;display:flex;align-items:center;justify-content:center;margin-bottom:16px;">
                        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
                    </div>
                    <h3>Children Adoption Home</h3>
                    <div style="font-size:14px;font-weight:700;color:var(--primary);margin-bottom:10px;">&#2970;&#3007;&#2993;&#2986;&#3021;&#2986;&#3009; &#2980;&#2980;&#3021;&#2980;&#3014;&#2975;&#3009;&#2980;&#3021;&#2980;&#2994;&#3021; &#2990;&#3016;&#2991;&#2990;&#3021;</div>
                    <p>Specialized adoption and infant care center providing certified pediatric healthcare, emotional bonding, and foster-care rehabilitation for infants and children.</p>
                </div>
            </div>
        </section>

        <!-- Compliance, Terms & Policies (Cashfree & Regulatory Compliance) -->
        <section id="policies" style="margin-bottom: 60px;">
            <div class="section-title">
                <h2>Trust Policies &amp; Compliance Terms</h2>
                <p>Governing transparency, donor privacy, and donation accounting</p>
            </div>

            <!-- Terms & Conditions -->
            <div class="policy-box" id="terms">
                <h3>Terms &amp; Conditions</h3>
                <p>By donating or using the online services of Nesakkaram Charitable Trust, you agree to the following terms:</p>
                <ul>
                    <li>All donations made to the trust are voluntary contributions aimed strictly at supporting charitable activities, including food distribution, orphan care, special education, and elderly rehabilitation.</li>
                    <li>Donors must provide true and accurate personal information (Name, Mobile, Email, and Address) to receive an official digital donation receipt and voucher.</li>
                    <li>The trust reserves the right to allocate donations to the most urgent humanitarian needs within its registered charitable objectives.</li>
                    <li>Digital payment receipts issued online serve as official proof of contribution for legal and accounting purposes.</li>
                </ul>
            </div>

            <!-- Privacy Policy -->
            <div class="policy-box" id="privacy">
                <h3>Privacy Policy</h3>
                <p>Nesakkaram Charitable Trust values your privacy and is dedicated to protecting your personal data:</p>
                <ul>
                    <li><strong>Information Collection:</strong> We collect only necessary donor details such as full name, phone number, email, address, and PAN number strictly for tax documentation, accounting audits, and receipt issuance.</li>
                    <li><strong>Payment Security:</strong> We do not store credit/debit card numbers, CVVs, or bank netbanking passwords. All online payments are handled securely through RBI-licensed payment gateways (Cashfree / Razorpay) using end-to-end 256-bit SSL encryption.</li>
                    <li><strong>No Third-Party Sharing:</strong> Your personal contact details are never rented, sold, or shared with commercial marketing agencies. Data is disclosed only to regulatory authorities (such as the Income Tax Department) as required by Indian law.</li>
                </ul>
            </div>

            <!-- Refund & Cancellation Policy -->
            <div class="policy-box" id="refund">
                <h3>Refund &amp; Cancellation Policy</h3>
                <p>Please review our refund and cancellation policy governing charitable donations:</p>
                <ul>
                    <li><strong>Voluntary Contributions:</strong> All donations contributed to Nesakkaram Charitable Trust are voluntary charitable gifts utilized directly for food, medicines, and daily resident care. Therefore, donations once processed are <strong>non-refundable and non-cancellable</strong>.</li>
                    <li><strong>Technical Discrepancies / Duplicate Deductions:</strong> In the rare event that an amount was debited more than once due to a banking network glitch or technical timeout, the donor should notify the trust via email at <strong>info@arminfo.in</strong> with transaction proofs (UTR / Order ID / Bank Statement) within <strong>7 working days</strong>.</li>
                    <li><strong>Resolution:</strong> Validated duplicate transactions will be verified against payment gateway settlement statements and reversed back to the donor's original payment source within 7–10 working days.</li>
                </ul>
            </div>
        </section>

        <!-- Contact Us Section -->
        <section id="contact">
            <div class="section-title">
                <h2>Contact Nesakkaram Charitable Trust</h2>
                <p>Reach out for donor queries, sponsorships, and institutional visits</p>
            </div>
            <div class="contact-grid">
                <div class="contact-card">
                    <h4>Official Email</h4>
                    <p><a href="mailto:info@arminfo.in">info@arminfo.in</a></p>
                </div>
                <div class="contact-card">
                    <h4>Helpline / Trust Phone</h4>
                    <p><a href="tel:+917373709961">+91 73737 09961</a></p>
                </div>
                <div class="contact-card">
                    <h4>Alternative Helpline</h4>
                    <p><a href="tel:+918825524702">+91 88255 24702</a></p>
                </div>
                <div class="contact-card">
                    <h4>Digital Technology Partner</h4>
                    <p>ARMSS INFO TECH</p>
                </div>
            </div>
        </section>

    </main>

    <!-- Footer -->
    <footer>
        <div class="footer-content">
            <div class="footer-links">
                <a href="#about">About Trust</a>
                <a href="#wings">Our Branches</a>
                <a href="#terms">Terms &amp; Conditions</a>
                <a href="#privacy">Privacy Policy</a>
                <a href="#refund">Refund Policy</a>
                <a href="#contact">Contact Us</a>
            </div>
            <p>© 2026 Nesakkaram Charitable Trust. All Rights Reserved. | Registered Charitable Trust under Indian Trusts Act.</p>
        </div>
    </footer>

</body>
</html>`

	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}
