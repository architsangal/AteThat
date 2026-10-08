# Nutrition target reference-source research (decision pending)

Phase 6 must not silently import a nutrition table or show an invented recommendation. The target domain model can record exact source attribution and preserve reference values separately from user overrides, but real calculation remains disabled until a source and method are approved.

## Most practical permissive starting point

The U.S. Food and Drug Administration (FDA) publishes [Daily Reference Values for food components](https://www.fda.gov/media/99059/download) and [Reference Daily Intakes for nutrients](https://www.fda.gov/media/99069/download). The [FDA website policy](https://www.fda.gov/about-fda/about-website/website-policies) says its content is generally public domain unless otherwise noted and may be reused; it recommends recording the copy date and linking back to the source. We found no contrary notice on these two FDA tables. Attribution should still name the FDA, include a source URL, and never imply FDA endorsement.

These are **food-label Daily Values**, not individual requirements. The adult/age-4-and-over table uses a 2,000-kcal reference diet. A Phase 6 implementation using these values should label them as general comparison benchmarks, not as a personalized calorie or nutrient prescription. In particular, the FDA 2,000-kcal reference must not be presented as a calculated energy target for every user.

## More personalized but not yet cleared for import

The [USDA National Agricultural Library DRI calculator](https://www.nal.usda.gov/human-nutrition-and-food-safety/dri-calculator) estimates calories and nutrient needs from age, sex, height, weight, and activity. Its [website policy](https://www.nal.usda.gov/web-policies-and-important-links) says most NAL content is public domain but warns that some material may be copyrighted or personal-use-only. The calculator says its DRI data come from the National Academies; it does not provide a clearly reusable, versioned data export or a license for embedding its complete calculation tables in AteThat. The [NIH Office of Dietary Supplements overview](https://ods.od.nih.gov/healthinformation/nutrientrecommendations.aspx) links DRI tables and explains that values vary by age and sex, but many linked source reports belong to the National Academies. Its [site policy](https://ods.od.nih.gov/About/Site_Policies.aspx) cautions that linked documents may be copyright-protected. We should not treat a public calculator as a blanket license to copy its underlying reference set.

The [ICMR-NIN Dietary Guidelines for Indians](https://www.nin.res.in/dietaryguidelines/pdfjs/locale/DGI24thJune2024fin.pdf) contain a restriction on storing or reproducing the publication electronically for a product without prior written permission. The [NIN RDA publication page](https://www.nin.res.in/RDA_Full_Report_2024.html) lists its detailed report as a paid publication. Do not import these tables into the application without appropriate permission and a review of the exact material used.

## Decision needed

1. Use FDA Daily Values as explicitly **general benchmarks** for an initial non-personalized view, while leaving personalized energy and nutrition targets unavailable; or
2. Obtain/approve a reusable personalized reference source and formula before enabling target calculation.

Either path must pin source/version, source URL, extraction/review date, covered population, calculation version, and explicit eligibility/unsupported cases. USDA FoodData Central remains the agreed source for **food composition**, which is separate from choosing a target-recommendation source.
