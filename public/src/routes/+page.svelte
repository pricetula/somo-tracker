<script>
	// Placeholder for any Svelte logic if needed later
</script>

<main class="container mx-auto px-4 py-8">
	<!-- Hero Section -->
	<section class="text-center py-16">
		<h1 class="text-5xl font-bold mb-4">Run Your School. Not a Spreadsheet.</h1>
		<p class="text-xl text-gray-700 mb-8">
			Students, fees, staff, and parents — one system, built CBC-first.
		</p>
		<button
			class="bg-blue-600 text-white px-8 py-3 rounded-lg text-lg font-semibold hover:bg-blue-700 transition-colors"
		>
			Join Waitlist
		</button>
	</section>

	<!-- The Hidden Cost of Doing It the Old Way -->
	<section class="py-16">
		<h2 class="text-4xl font-bold text-center mb-12">The Hidden Cost of Doing It the Old Way</h2>
		<div class="grid md:grid-cols-3 gap-8 text-center">
			<div>
				<h3 class="text-2xl font-semibold mb-2">PAPER TRAIL</h3>
				<p class="text-lg">hours/term</p>
				<p class="text-gray-600">Lost chasing fee balances, admission forms, and student files</p>
			</div>
			<div>
				<h3 class="text-2xl font-semibold mb-2">SCATTERED SYSTEMS</h3>
				<p class="text-lg">5+ tools</p>
				<p class="text-gray-600">
					Spreadsheets, WhatsApp, and paper registers doing the job of one platform
				</p>
			</div>
			<div>
				<h3 class="text-2xl font-semibold mb-2">SLOW ANSWERS</h3>
				<p class="text-lg">days, not minutes</p>
				<p class="text-gray-600">
					Before a parent or admin gets a straight answer on fees or attendance
				</p>
			</div>
		</div>
		<p class="text-center text-lg mt-12 italic">But it doesn't have to be this way...</p>
	</section>

	<!-- The Cost of Chaos vs. Clarity (Interactive Calculator Section) -->
	<section class="bg-gray-50 py-16">
		<h2 class="text-4xl font-bold text-center mb-8">The Cost of Chaos vs. Clarity</h2>
		<p class="text-center text-xl mb-8">
			"Is your school still running fees, enrollment, and records across five different tools?"
		</p>

		<div class="max-w-xl mx-auto bg-white p-8 rounded-lg shadow-md">
			<div class="mb-4">
				<label for="totalStudents" class="block text-gray-700 text-sm font-bold mb-2"
					>Total Students:</label
				>
				<input
					type="number"
					id="totalStudents"
					value="400"
					class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
				/>
			</div>
			<div class="mb-4">
				<label for="staffMembers" class="block text-gray-700 text-sm font-bold mb-2"
					>Staff Members:</label
				>
				<input
					type="number"
					id="staffMembers"
					value="30"
					class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
				/>
			</div>
			<div class="mb-6">
				<label for="parentInquiries" class="block text-gray-700 text-sm font-bold mb-2"
					>Parent Inquiries per Week:</label
				>
				<input
					type="number"
					id="parentInquiries"
					value="50"
					class="shadow appearance-none border rounded w-full py-2 px-3 text-gray-700 leading-tight focus:outline-none focus:shadow-outline"
				/>
			</div>

			<div class="text-center text-lg font-semibold mb-6">
				<p>Admin Hours Saved: <span class="text-green-600">0 hours/term</span></p>
				<p>Fee Reconciliation Time Cut: <span class="text-green-600">0%</span></p>
				<p>SomoTracker Efficiency Gain: <span class="text-green-600">92%</span></p>
			</div>

			<div class="text-center">
				<p class="mb-4">Ready to bring your whole school onto one platform?</p>
				<button
					class="bg-blue-600 text-white px-8 py-3 rounded-lg text-lg font-semibold hover:bg-blue-700 transition-colors"
				>
					Join Waitlist
				</button>
			</div>
		</div>
	</section>

	<!-- From Chaos to Clarity in 3 Steps -->
	<section class="py-16">
		<h2 class="text-4xl font-bold text-center mb-12">From Chaos to Clarity in 3 Steps</h2>
		<div class="grid md:grid-cols-3 gap-8">
			<div class="text-center">
				<h3 class="text-2xl font-semibold mb-2">1 — 15 minutes: Simple Data Upload</h3>
				<ul class="list-disc list-inside text-gray-700">
					<li>CSV or direct import</li>
					<li>We handle messy data, UPI numbers and all</li>
				</ul>
			</div>
			<div class="text-center">
				<h3 class="text-2xl font-semibold mb-2">2 — 1 hour (automated): School Setup</h3>
				<ul class="list-disc list-inside text-gray-700">
					<li>Grades, streams, and terms configured for CBC/CBE</li>
					<li>Zero manual setup required</li>
				</ul>
			</div>
			<div class="text-center">
				<h3 class="text-2xl font-semibold mb-2">3 — Instant: Total School Visibility</h3>
				<ul class="list-disc list-inside text-gray-700">
					<li>Students, fees, staff, and parents — all in one dashboard</li>
					<li>Every stakeholder gets the right access, instantly</li>
				</ul>
			</div>
		</div>
	</section>

	<!-- What's Actually Inside -->
	<section class="bg-blue-700 text-white py-16">
		<h2 class="text-4xl font-bold text-center mb-12">What's Actually Inside</h2>
		<div class="max-w-2xl mx-auto grid md:grid-cols-1 gap-6">
			<div class="flex items-start">
				<span class="text-3xl mr-4">🎓</span>
				<div>
					<h3 class="text-2xl font-semibold mb-1">CBC-Native, Not Bolted On</h3>
					<p>UPI, NEMIS, TSC, KNEC cycles built in from day one</p>
				</div>
			</div>
			<div class="flex items-start">
				<span class="text-3xl mr-4">👨‍👩‍👧</span>
				<div>
					<h3 class="text-2xl font-semibold mb-1">Parents, Included</h3>
					<p>direct access to fees and student info, no more WhatsApp chaos</p>
				</div>
			</div>
			<div class="flex items-start">
				<span class="text-3xl mr-4">💳</span>
				<div>
					<h3 class="text-2xl font-semibold mb-1">Fees That Reconcile Themselves</h3>
					<p>M-Pesa reference matching, real payment status</p>
				</div>
			</div>
			<div class="flex items-start">
				<span class="text-3xl mr-4">🏥</span>
				<div>
					<h3 class="text-2xl font-semibold mb-1">Health Records That Matter</h3>
					<p>allergies, conditions, and emergency info at a glance</p>
				</div>
			</div>
			<div class="flex items-start">
				<span class="text-3xl mr-4">🧑‍🏫</span>
				<div>
					<h3 class="text-2xl font-semibold mb-1">Staff & Roles, Sorted</h3>
					<p>admins, teachers, nurses, finance — right access, no fuss</p>
				</div>
			</div>
		</div>
	</section>

	<!-- Final CTA -->
	<section class="text-center py-16">
		<h2 class="text-4xl font-bold mb-4">Your School Deserves a System That Gets It.</h2>
		<p class="text-xl text-gray-700 mb-8">
			Join the schools moving off spreadsheets and onto SomoTracker.
		</p>
		<button
			class="bg-blue-600 text-white px-8 py-3 rounded-lg text-lg font-semibold hover:bg-blue-700 transition-colors mb-4"
		>
			Join Waitlist
		</button>
		<p class="text-sm text-gray-500">
			Only 25 spots left · 60-day free pilot · No credit card required
		</p>
	</section>
</main>

<footer class="bg-gray-800 text-white text-center py-8">
	<div class="container mx-auto">
		<p class="mb-2">SomoTracker &copy; 2024</p>
		<!-- One-liner Taglines -->
		<ul class="list-none p-0 text-sm text-gray-400">
			<li>"Built for CBC. Not adapted for it."</li>
			<li>"One school. One system. Zero chaos."</li>
			<li>"Fees, students, staff, parents — sorted."</li>
			<li>"The school platform that already speaks CBC."</li>
			<li>"Less admin. More school."</li>
		</ul>
	</div>
</footer>

<style lang="postcss">
	/* Add any global styles here, or component-specific styles */
	:global(html, body) {
		@apply h-full overflow-x-hidden;
	}
	:global(body) {
		@apply flex min-h-screen flex-col;
	}
	main {
		@apply flex-grow;
	}
</style>
