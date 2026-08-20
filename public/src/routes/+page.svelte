<script lang="ts">
	import { Check } from '@lucide/svelte';
	import Container from '$lib/components/shared/container/container.svelte';
	import JoinWaitList from '$lib/components/pages/landing/JoinWaitList.svelte';
	import ValueShowcase from '$lib/components/pages/landing/ValueShowcase.svelte';
</script>

<article class="mt-18 md:mt-30">
	<Container>
		<section id="hero" class="grid min-h-96 gap-8 md:grid-cols-2">
			<div class="order-2 md:order-1">
				<h1 class="mb-4 text-2xl leading-tight font-bold tracking-tight">
					<span class="hero-line hero-line-2">Zero Waste.</span>
					<br />
					<span class="hero-line hero-line-1">Zero Printing.</span>
					<br />
					<span class="hero-line hero-line-3"> 100% Academic Insight.</span>
				</h1>

				<p class="hero-p mb-4">
					Eliminate administrative overhead and printing costs. Harness real-time academic
					data insights to identify struggling students early and guide them toward a
					successful, personalized learning path.
				</p>

				<ul class="mb-8 space-y-1 text-xs">
					<li class="hero-check hero-check-1 flex items-center gap-1">
						<Check size="14" class="text-green-500" />
						<span>Assessments</span>
					</li>
					<li class="hero-check hero-check-2 flex items-center gap-1">
						<Check size="14" class="text-green-500" />
						<span>AI-Driven Gap Forecasting</span>
					</li>
					<li class="hero-check hero-check-3 flex items-center gap-1">
						<Check size="14" class="text-green-500" />
						<span>Setup in &lt; 2 hours</span>
					</li>
				</ul>

				<div class="hero-cta">
					<JoinWaitList />
				</div>
			</div>
			<div class="order-1 md:order-2">
				<img src="/cloud.svg" alt="hero" class="w-9/12" />
			</div>
		</section>

		<ValueShowcase />
	</Container>
</article>
